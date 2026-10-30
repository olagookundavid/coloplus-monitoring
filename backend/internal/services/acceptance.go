package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// AcceptanceService runs a site's commissioning and acceptance: Coloplus
// builds and submits a dossier, Airtel inspects and accepts or rejects. The
// two sides are kept apart, and only Airtel's own decision counts as
// acceptance. It is reported separately from operational performance.
type AcceptanceService struct {
	acceptances ports.AcceptanceRepository
	sites       ports.SiteRepository
	users       ports.UserRepository
	tx          ports.TxManager
	audit       *AuditService
	clock       ports.Clock
	notifier    *NotificationService
	loc         *time.Location
	baseURL     string
}

func NewAcceptanceService(
	acceptances ports.AcceptanceRepository, sites ports.SiteRepository, users ports.UserRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, notifier *NotificationService,
	loc *time.Location, baseURL string,
) *AcceptanceService {
	return &AcceptanceService{
		acceptances: acceptances, sites: sites, users: users, tx: tx, audit: audit, clock: clock,
		notifier: notifier, loc: loc, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *AcceptanceService) link(siteID string) string {
	return s.baseURL + "/sites/" + siteID + "/acceptance"
}

// AcceptanceView is a process with its history.
type AcceptanceView struct {
	Acceptance domain.Acceptance
	Events     []domain.AcceptanceEvent
}

type RequestAcceptanceInput struct {
	SiteID     string
	NextAction string
	DueAt      *time.Time
}

type ListAcceptancesInput struct {
	Stages   []domain.AcceptanceStage
	OpenOnly bool
	Overdue  bool
	Limit    int
	Offset   int
}

// loadSite hides a site from people outside their scope.
func (s *AcceptanceService) load(ctx context.Context, actor domain.Actor, siteID string, forUpdate bool) (domain.Acceptance, error) {
	if !actor.Can(domain.PermAcceptanceRead) {
		return domain.Acceptance{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.Acceptance{}, apperrors.ErrNotFound
	}
	return s.acceptances.GetBySite(ctx, siteID, forUpdate)
}

func (s *AcceptanceService) event(actor domain.Actor, action string, a domain.Acceptance, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{"site_id": a.SiteID, "stage": string(a.Stage)}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "acceptance", EntityID: a.ID, Metadata: meta}
}

// Request starts the acceptance process for a site still pending acceptance.
func (s *AcceptanceService) Request(ctx context.Context, actor domain.Actor, in RequestAcceptanceInput) (domain.Acceptance, error) {
	if !actor.Can(domain.PermAcceptanceManage) {
		return domain.Acceptance{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(in.SiteID) {
		return domain.Acceptance{}, apperrors.ErrNotFound
	}
	now := s.clock.Now()
	if in.DueAt != nil && !in.DueAt.After(now) {
		return domain.Acceptance{}, invalid("the due date must be in the future")
	}
	a := domain.Acceptance{
		SiteID: in.SiteID, Responsible: domain.PartyColoplus, NextAction: strings.TrimSpace(in.NextAction),
		DueAt: in.DueAt, RequestedAt: now, CreatedBy: actor.UserID,
	}

	var created domain.Acceptance
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		site, err := s.sites.GetByID(ctx, in.SiteID)
		if err != nil {
			return err
		}
		if site.Status != domain.SitePendingAcceptance {
			return conflict("only a site pending acceptance can enter acceptance")
		}
		if created, err = s.acceptances.Create(ctx, a); err != nil {
			return err
		}
		if _, err := s.acceptances.AddEvent(ctx, domain.AcceptanceEvent{
			AcceptanceID: created.ID, AuthorID: actor.UserID, Party: domain.PartyColoplus,
			ToStage: domain.StageRequested, Body: "Acceptance requested", CreatedAt: now,
		}); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, s.event(actor, "acceptance.requested", created, nil))
	})
	return created, err
}

// Get returns a site's acceptance with its history.
func (s *AcceptanceService) Get(ctx context.Context, actor domain.Actor, siteID string) (AcceptanceView, error) {
	a, err := s.load(ctx, actor, siteID, false)
	if err != nil {
		return AcceptanceView{}, err
	}
	events, err := s.acceptances.Events(ctx, a.ID)
	if err != nil {
		return AcceptanceView{}, err
	}
	return AcceptanceView{Acceptance: a, Events: events}, nil
}

// List is the commissioning backlog for the caller's scope.
func (s *AcceptanceService) List(ctx context.Context, actor domain.Actor, in ListAcceptancesInput) ([]domain.Acceptance, int, error) {
	if !actor.Can(domain.PermAcceptanceRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := ports.AcceptanceFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Stages: in.Stages, OpenOnly: in.OpenOnly,
	}
	if in.Overdue {
		now := s.clock.Now()
		f.OverdueAt = &now
	}
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	return s.acceptances.List(ctx, f)
}

// Transition moves the process along. Repeating a step that already happened
// changes nothing.
//
//	Coloplus: requested -> built -> ready for inspection (with the dossier); rejected -> built
//	Airtel:   ready for inspection -> inspected | rejected; inspected -> accepted | rejected
//	Either:   hold, and resume to where it was
//
// When Airtel accepts, a site still pending acceptance becomes active, which
// is what brings it into the service KPIs.
func (s *AcceptanceService) Transition(ctx context.Context, actor domain.Actor, siteID string, step domain.AcceptanceStep) (domain.Acceptance, error) {
	party, ok := domain.AcceptanceParty(actor)
	if !ok {
		return domain.Acceptance{}, apperrors.ErrForbidden
	}
	if !domain.ValidAcceptanceStage(step.To) || step.To == domain.StageRequested {
		return domain.Acceptance{}, invalid("choose a valid next stage")
	}

	var out domain.Acceptance
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		a, err := s.load(ctx, actor, siteID, true)
		if err != nil {
			return err
		}
		if a.Stage == step.To {
			out = a
			return nil
		}
		if !a.CanMove(party, step.To) {
			return conflict("an acceptance that is %s cannot move to %s by %s", stageWord(a.Stage), stageWord(step.To), party)
		}
		now := s.clock.Now()
		if problems := a.CheckStep(party, step, now); len(problems) > 0 {
			return invalidProblems(problems)
		}
		from := a.Stage
		a.Apply(party, actor.UserID, step, now)
		e, err := s.acceptances.AddEvent(ctx, domain.AcceptanceEvent{
			AcceptanceID: a.ID, AuthorID: actor.UserID, Party: party, FromStage: from, ToStage: a.Stage,
			Body: strings.TrimSpace(step.Note), CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if err := s.acceptances.Save(ctx, a); err != nil {
			return err
		}
		if err := s.audit.RecordInTx(ctx, s.event(actor, "acceptance."+string(a.Stage), a, map[string]any{"from": string(from)})); err != nil {
			return err
		}
		if a.Stage == domain.StageAccepted {
			if err := s.activate(ctx, actor, a, now); err != nil {
				return err
			}
		}
		out = a
		return s.notifyOtherParty(ctx, a, e)
	})
	return out, err
}

// activate brings an accepted site into service. A suspended site is left as
// it is: suspending is its own decision.
func (s *AcceptanceService) activate(ctx context.Context, actor domain.Actor, a domain.Acceptance, now time.Time) error {
	site, err := s.sites.GetByID(ctx, a.SiteID)
	if err != nil {
		return err
	}
	if site.Status != domain.SitePendingAcceptance {
		return nil
	}
	site.Status = domain.SiteActive
	if site.CommissioningDate == nil {
		y, m, d := now.In(s.loc).Date()
		date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		site.CommissioningDate = &date
	}
	if err := s.sites.Update(ctx, site); err != nil {
		return err
	}
	return s.audit.RecordInTx(ctx, domain.AuditEvent{
		ActorID: actor.UserID, Action: "site.activated", EntityType: "site", EntityID: site.ID,
		Metadata: map[string]any{"site_id": site.ID, "acceptance_id": a.ID},
	})
}

// notifyOtherParty emails the other side, scoped to the site. Internal
// progress (built, or rework after a rejection) is Coloplus's own business.
func (s *AcceptanceService) notifyOtherParty(ctx context.Context, a domain.Acceptance, e domain.AcceptanceEvent) error {
	if e.ToStage == domain.StageBuilt {
		return nil
	}
	perm := domain.PermAcceptanceManage
	if e.Party == domain.PartyColoplus {
		perm = domain.PermAcceptanceReview
	}
	recipients, err := s.users.UsersWithPermission(ctx, perm, a.SiteID)
	if err != nil || len(recipients) == 0 {
		return err
	}
	site, err := s.sites.GetByID(ctx, a.SiteID)
	if err != nil {
		return err
	}
	subject, body := acceptanceEventEmail(site, e, s.link(a.SiteID))
	_, err = s.notifier.EnqueueFor(ctx, recipients, "acceptance.event", "acceptance-event:"+e.ID, subject, body)
	return err
}

const acceptanceAlertLimit = 500

// AlertOverdue emails whoever the next step is waiting on, and Coloplus's
// supervisors, about every acceptance whose due time has passed. Each missed
// due time is alerted once. A hold stops the clock. It returns how many
// emails it queued.
func (s *AcceptanceService) AlertOverdue(ctx context.Context) (int, error) {
	now := s.clock.Now()
	late, _, err := s.acceptances.List(ctx, ports.AcceptanceFilter{AllSites: true, OverdueAt: &now, Limit: acceptanceAlertLimit})
	if err != nil {
		return 0, err
	}
	queued := 0
	for _, a := range late {
		if a.DueAt == nil {
			continue
		}
		site, err := s.sites.GetByID(ctx, a.SiteID)
		if err != nil {
			return queued, err
		}
		recipients, err := s.users.UsersWithPermission(ctx, domain.PermAcceptanceManage, a.SiteID)
		if err != nil {
			return queued, err
		}
		if a.Responsible == domain.PartyAirtel {
			airtel, err := s.users.UsersWithPermission(ctx, domain.PermAcceptanceReview, a.SiteID)
			if err != nil {
				return queued, err
			}
			for _, u := range airtel {
				recipients = appendUnique(recipients, u)
			}
		}
		subject, body := acceptanceOverdueEmail(site, a, *a.DueAt, s.loc, s.link(a.SiteID))
		key := fmt.Sprintf("acceptance-overdue:%s:%d", a.ID, a.DueAt.Unix())
		n, err := s.notifier.EnqueueFor(ctx, recipients, "acceptance.overdue", key, subject, body)
		queued += n
		if err != nil {
			return queued, err
		}
	}
	return queued, nil
}
