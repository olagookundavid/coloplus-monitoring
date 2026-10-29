package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// DisputeService runs the dispute workflow between Airtel's reviewers and
// Coloplus: raise, review, ask for evidence, agree a correction, resolve,
// acknowledge. Coloplus's approval and Airtel's agreement are separate acts,
// and neither changes the original operational record.
type DisputeService struct {
	disputes ports.DisputeRepository
	reports  ports.ReportRepository
	sites    ports.SiteRepository
	users    ports.UserRepository
	tx       ports.TxManager
	audit    *AuditService
	clock    ports.Clock
	notifier *NotificationService
	loc      *time.Location
	baseURL  string
}

func NewDisputeService(
	disputes ports.DisputeRepository, reports ports.ReportRepository, sites ports.SiteRepository,
	users ports.UserRepository, tx ports.TxManager, audit *AuditService, clock ports.Clock,
	notifier *NotificationService, loc *time.Location, baseURL string,
) *DisputeService {
	return &DisputeService{
		disputes: disputes, reports: reports, sites: sites, users: users, tx: tx, audit: audit,
		clock: clock, notifier: notifier, loc: loc, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *DisputeService) link(d domain.Dispute) string { return s.baseURL + "/disputes/" + d.ID }

// DisputeView is a dispute with the history the caller may see.
type DisputeView struct {
	Dispute domain.Dispute
	Events  []domain.DisputeEvent
}

type DisputeInput struct {
	SiteID      string
	ReportID    string
	IntervalID  string
	Metric      domain.DisputeMetric
	PeriodFrom  *time.Time
	PeriodTo    *time.Time
	Category    domain.DisputeCategory
	Title       string
	Description string
}

type ListDisputesInput struct {
	SiteID   string
	ReportID string
	Statuses []domain.DisputeStatus
	OpenOnly bool
	Limit    int
	Offset   int
}

// IsStaff says whether the caller sees Coloplus's side of a dispute: who owns
// it and the internal notes. Airtel users never do.
func (s *DisputeService) IsStaff(actor domain.Actor) bool {
	return !actor.Can(domain.PermPublishedRead)
}

func (s *DisputeService) scope(actor domain.Actor) ports.DisputeFilter {
	return ports.DisputeFilter{AllSites: actor.AllSites, SiteIDs: actor.SiteIDList()}
}

// load fetches a dispute and hides it from people outside its site scope.
func (s *DisputeService) load(ctx context.Context, actor domain.Actor, id string, forUpdate bool) (domain.Dispute, error) {
	if !actor.Can(domain.PermDisputesRead) {
		return domain.Dispute{}, apperrors.ErrForbidden
	}
	d, err := s.disputes.GetByID(ctx, id, forUpdate)
	if err != nil {
		return domain.Dispute{}, err
	}
	if !actor.CanAccessSite(d.SiteID) {
		return domain.Dispute{}, apperrors.ErrNotFound
	}
	return d, nil
}

func (s *DisputeService) event(actor domain.Actor, action string, d domain.Dispute, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{"site_id": d.SiteID, "number": d.Number}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "dispute", EntityID: d.ID, Metadata: meta}
}

// Raise opens a dispute about a site, a published report, or one interval of
// it. Only an approved report can be disputed, because that is all Airtel sees.
func (s *DisputeService) Raise(ctx context.Context, actor domain.Actor, in DisputeInput) (domain.Dispute, error) {
	if !actor.Can(domain.PermDisputesRaise) {
		return domain.Dispute{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(in.SiteID) {
		return domain.Dispute{}, apperrors.ErrNotFound
	}
	now := s.clock.Now()
	d := domain.Dispute{
		SiteID: in.SiteID, ReportID: in.ReportID, IntervalID: in.IntervalID, Metric: in.Metric,
		PeriodFrom: in.PeriodFrom, PeriodTo: in.PeriodTo, Category: in.Category,
		Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description),
		RaisedBy: actor.UserID, CreatedAt: now,
	}
	y, m, day := now.In(s.loc).Date()
	if problems := d.ValidateNew(time.Date(y, m, day, 0, 0, 0, 0, time.UTC)); len(problems) > 0 {
		return domain.Dispute{}, invalidProblems(problems)
	}

	var created domain.Dispute
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.sites.GetByID(ctx, in.SiteID); err != nil {
			return err
		}
		if err := s.checkTarget(ctx, d); err != nil {
			return err
		}
		var err error
		if created, err = s.disputes.Create(ctx, d); err != nil {
			return err
		}
		e, err := s.disputes.AddEvent(ctx, domain.DisputeEvent{
			DisputeID: created.ID, AuthorID: actor.UserID, Party: domain.PartyAirtel, Kind: domain.EventStatus,
			ToStatus: domain.DisputeOpen, Body: created.Description, CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if err := s.audit.RecordInTx(ctx, s.event(actor, "dispute.raised", created, map[string]any{"category": string(created.Category)})); err != nil {
			return err
		}
		return s.notifyOtherParty(ctx, created, e)
	})
	return created, err
}

// checkTarget makes sure what is disputed is a published report of this site
// and, if named, one of its intervals.
func (s *DisputeService) checkTarget(ctx context.Context, d domain.Dispute) error {
	if d.ReportID == "" {
		return nil
	}
	r, err := s.reports.GetByID(ctx, d.ReportID, false)
	if err != nil {
		if isNotFound(err) {
			return invalid("that report does not exist")
		}
		return err
	}
	if r.SiteID != d.SiteID || r.Status != domain.ReportApproved {
		return invalid("only a published report of this site can be disputed")
	}
	if d.IntervalID == "" {
		return nil
	}
	for _, iv := range r.Intervals {
		if iv.ID == d.IntervalID {
			return nil
		}
	}
	return invalid("that interval is not part of this report")
}

// Get returns a dispute with its history. Airtel does not see Coloplus's
// internal notes.
func (s *DisputeService) Get(ctx context.Context, actor domain.Actor, id string) (DisputeView, error) {
	d, err := s.load(ctx, actor, id, false)
	if err != nil {
		return DisputeView{}, err
	}
	events, err := s.disputes.Events(ctx, id, !s.IsStaff(actor))
	if err != nil {
		return DisputeView{}, err
	}
	return DisputeView{Dispute: d, Events: events}, nil
}

func (s *DisputeService) List(ctx context.Context, actor domain.Actor, in ListDisputesInput) ([]domain.Dispute, int, error) {
	if !actor.Can(domain.PermDisputesRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := s.scope(actor)
	f.SiteID, f.ReportID, f.Statuses, f.OpenOnly = in.SiteID, in.ReportID, in.Statuses, in.OpenOnly
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	if f.SiteID != "" && !actor.CanAccessSite(f.SiteID) {
		return []domain.Dispute{}, 0, nil
	}
	return s.disputes.List(ctx, f)
}

func (s *DisputeService) Summary(ctx context.Context, actor domain.Actor) (ports.DisputeCounts, error) {
	if !actor.Can(domain.PermDisputesRead) {
		return ports.DisputeCounts{}, apperrors.ErrForbidden
	}
	return s.disputes.Counts(ctx, s.scope(actor))
}

// OpenByReport says, for reports the caller can already see, how many
// disputes about each have not closed, so a disputed figure always shows it.
func (s *DisputeService) OpenByReport(ctx context.Context, actor domain.Actor, reportIDs []string) (map[string]int, error) {
	if !actor.Can(domain.PermDisputesRead) {
		return map[string]int{}, nil
	}
	return s.disputes.OpenByReport(ctx, reportIDs)
}

// Comment adds to the conversation. Coloplus may mark a comment internal, so
// Airtel never sees it. An Airtel comment on a dispute waiting for their
// evidence hands it back to Coloplus.
func (s *DisputeService) Comment(ctx context.Context, actor domain.Actor, id, body string, internal bool) (domain.DisputeEvent, error) {
	party, ok := domain.PartyOf(actor)
	if !ok {
		return domain.DisputeEvent{}, apperrors.ErrForbidden
	}
	if internal && party != domain.PartyColoplus {
		return domain.DisputeEvent{}, apperrors.ErrForbidden
	}
	if problems := domain.ValidateComment(body); len(problems) > 0 {
		return domain.DisputeEvent{}, invalidProblems(problems)
	}

	var out domain.DisputeEvent
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		d, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if !d.Open() {
			return conflict("this dispute is closed")
		}
		now := s.clock.Now()
		out, err = s.disputes.AddEvent(ctx, domain.DisputeEvent{
			DisputeID: id, AuthorID: actor.UserID, Party: party, Kind: domain.EventComment,
			FromStatus: d.Status, ToStatus: d.Status, Body: strings.TrimSpace(body), Internal: internal, CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if !internal {
			if err := s.notifyOtherParty(ctx, d, out); err != nil {
				return err
			}
		}
		if party == domain.PartyAirtel && d.Status == domain.DisputeAwaitingEvidence {
			return s.move(ctx, actor, party, d, domain.DisputeStep{To: domain.DisputeUnderReview, Note: "Evidence provided"}, false)
		}
		return nil
	})
	return out, err
}

// Transition moves a dispute along its workflow. Repeating a step that has
// already happened changes nothing, so a retry is safe.
//
//	Coloplus: open -> under review -> awaiting evidence | agreed correction | resolved
//	Airtel:   withdraw (closes it), give evidence or reopen (back to under review),
//	          acknowledge a resolution (closes it)
func (s *DisputeService) Transition(ctx context.Context, actor domain.Actor, id string, step domain.DisputeStep) (domain.Dispute, error) {
	party, ok := domain.PartyOf(actor)
	if !ok {
		return domain.Dispute{}, apperrors.ErrForbidden
	}
	if !domain.ValidDisputeStatus(step.To) || step.To == domain.DisputeOpen {
		return domain.Dispute{}, invalid("choose a valid next status")
	}

	var out domain.Dispute
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		d, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if d.Status == step.To {
			out = d
			return nil
		}
		if !d.CanMove(party, step.To) {
			return conflict("a dispute that is %s cannot be moved to %s by %s",
				disputeStatusWord(d.Status), disputeStatusWord(step.To), party)
		}
		if problems := d.CheckStep(party, step); len(problems) > 0 {
			return invalidProblems(problems)
		}
		if step.To == domain.DisputeResolved && step.Outcome == domain.OutcomeUpheld {
			if err := s.checkCorrection(ctx, d, step.ResolutionReportID); err != nil {
				return err
			}
		}
		if err := s.move(ctx, actor, party, d, step, true); err != nil {
			return err
		}
		out, err = s.disputes.GetByID(ctx, id, false)
		return err
	})
	return out, err
}

// move applies a step inside the caller's transaction: history, state, audit
// and the email to the other party.
func (s *DisputeService) move(ctx context.Context, actor domain.Actor, party domain.DisputeParty, d domain.Dispute, step domain.DisputeStep, announce bool) error {
	now := s.clock.Now()
	from := d.Status
	d.Apply(party, actor.UserID, step, now)
	e, err := s.disputes.AddEvent(ctx, domain.DisputeEvent{
		DisputeID: d.ID, AuthorID: actor.UserID, Party: party, Kind: domain.EventStatus,
		FromStatus: from, ToStatus: d.Status, Body: strings.TrimSpace(step.Note), CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if err := s.disputes.Save(ctx, d); err != nil {
		return err
	}
	extra := map[string]any{"from": string(from)}
	if d.Outcome != domain.OutcomeNone {
		extra["outcome"] = string(d.Outcome)
	}
	if err := s.audit.RecordInTx(ctx, s.event(actor, "dispute."+string(d.Status), d, extra)); err != nil {
		return err
	}
	if !announce {
		return nil
	}
	return s.notifyOtherParty(ctx, d, e)
}

// checkCorrection makes sure an upheld dispute is settled by a real, approved
// correction: a later revision of the same site and day.
func (s *DisputeService) checkCorrection(ctx context.Context, d domain.Dispute, reportID string) error {
	if reportID == "" {
		return nil
	}
	fix, err := s.reports.GetByID(ctx, reportID, false)
	if err != nil {
		if isNotFound(err) {
			return invalid("the corrected report does not exist")
		}
		return err
	}
	if fix.SiteID != d.SiteID || fix.Status != domain.ReportApproved {
		return invalid("the corrected report must be an approved report of this site")
	}
	if d.ReportID == "" {
		return nil
	}
	orig, err := s.reports.GetByID(ctx, d.ReportID, false)
	if err != nil {
		return err
	}
	if !sameDay(fix.Date, orig.Date) || fix.Revision <= orig.Revision {
		return invalid("the corrected report must be a later revision of the disputed day")
	}
	return nil
}

func isNotFound(err error) bool { return errors.Is(err, apperrors.ErrNotFound) }

func sameDay(a, b time.Time) bool { return a.Format("2006-01-02") == b.Format("2006-01-02") }

// notifyOtherParty emails the people on the other side of the conversation,
// scoped to the site. Internal notes never come here. It queues in the
// caller's transaction, one email per person and event.
func (s *DisputeService) notifyOtherParty(ctx context.Context, d domain.Dispute, e domain.DisputeEvent) error {
	perm := domain.PermDisputesManage
	if e.Party == domain.PartyColoplus {
		perm = domain.PermDisputesRaise
	}
	recipients, err := s.users.UsersWithPermission(ctx, perm, d.SiteID)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}
	site, err := s.sites.GetByID(ctx, d.SiteID)
	if err != nil {
		return err
	}
	subject, body := disputeEventEmail(d, site, e, s.link(d))
	_, err = s.notifier.EnqueueFor(ctx, recipients, "dispute.event", fmt.Sprintf("dispute-event:%s", e.ID), subject, body)
	return err
}
