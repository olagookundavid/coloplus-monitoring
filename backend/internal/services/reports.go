package services

import (
	"context"
	"errors"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// ReportService runs the daily reporting workflow:
// draft -> submitted -> approved, with rejection and linked corrections.
type ReportService struct {
	reports ports.ReportRepository
	sites   ports.SiteRepository
	tx      ports.TxManager
	audit   *AuditService
	clock   ports.Clock
	loc     *time.Location
}

func NewReportService(
	reports ports.ReportRepository, sites ports.SiteRepository, tx ports.TxManager,
	audit *AuditService, clock ports.Clock, loc *time.Location,
) *ReportService {
	return &ReportService{reports: reports, sites: sites, tx: tx, audit: audit, clock: clock, loc: loc}
}

// Calculation records how a derived figure was produced, so the "How
// calculated" panel can show it. Inputs are the report revision used.
type Calculation struct {
	Formula    string
	Version    string
	ReportID   string
	Revision   int
	From, To   time.Time
	ComputedAt time.Time
	Warnings   []string
}

// ReportView is a report with its site and computed summary.
type ReportView struct {
	Report      domain.DailyReport
	Site        domain.Site
	Summary     domain.DailySummary
	Calculation Calculation
}

const (
	formulaDailyAvailability = "daily-availability"
	formulaVersion           = "1"
)

type ReportInput struct {
	Notes     string
	Intervals []domain.PowerInterval
}

// today returns the current reporting date in the reporting time zone.
func (s *ReportService) today() time.Time {
	y, m, d := s.clock.Now().In(s.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ParseDate reads a YYYY-MM-DD reporting date.
func ParseDate(v string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return time.Time{}, invalid("dates must look like 2026-12-01")
	}
	return t, nil
}

func (s *ReportService) CreateDraft(ctx context.Context, actor domain.Actor, siteID string, date time.Time, in ReportInput) (domain.DailyReport, error) {
	if !actor.Can(domain.PermReportsCreate) {
		return domain.DailyReport{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.DailyReport{}, apperrors.ErrNotFound
	}
	site, err := s.sites.GetByID(ctx, siteID)
	if err != nil {
		return domain.DailyReport{}, err
	}
	if site.Status == domain.SiteSuspended {
		return domain.DailyReport{}, invalid("this site is suspended and cannot receive reports")
	}
	if date.After(s.today()) {
		return domain.DailyReport{}, invalid("a report cannot be for a future date")
	}
	if problems := domain.ValidateIntervals(site, date, in.Intervals, s.clock.Now(), s.loc); len(problems) > 0 {
		return domain.DailyReport{}, invalidProblems(problems)
	}

	var created domain.DailyReport
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.reports.FindApproved(ctx, siteID, date); err == nil {
			return conflict("this day already has an approved report; start a correction instead")
		} else if !errors.Is(err, apperrors.ErrNotFound) {
			return err
		}
		rev, err := s.reports.NextRevision(ctx, siteID, date)
		if err != nil {
			return err
		}
		created, err = s.reports.Create(ctx, domain.DailyReport{
			SiteID: siteID, Date: date, Revision: rev, Status: domain.ReportDraft,
			AuthorID: actor.UserID, Notes: in.Notes, Intervals: in.Intervals,
		})
		return err
	})
	return created, err
}

// load fetches a report and applies visibility: people outside the site's
// scope, and Airtel users looking at anything unapproved, get "not found".
func (s *ReportService) load(ctx context.Context, actor domain.Actor, id string, forUpdate bool) (domain.DailyReport, error) {
	r, err := s.reports.GetByID(ctx, id, forUpdate)
	if err != nil {
		return domain.DailyReport{}, err
	}
	if !actor.CanAccessSite(r.SiteID) || !canSee(actor, r) {
		return domain.DailyReport{}, apperrors.ErrNotFound
	}
	return r, nil
}

func canSee(actor domain.Actor, r domain.DailyReport) bool {
	if actor.Can(domain.PermReportsRead) {
		return true
	}
	return actor.Can(domain.PermPublishedRead) && r.Status == domain.ReportApproved
}

func (s *ReportService) SaveDraft(ctx context.Context, actor domain.Actor, id string, rowVersion int, in ReportInput) (domain.DailyReport, error) {
	if !actor.Can(domain.PermReportsCreate) {
		return domain.DailyReport{}, apperrors.ErrForbidden
	}
	var saved domain.DailyReport
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if r.AuthorID != actor.UserID {
			return apperrors.ErrForbidden
		}
		if !r.Editable() {
			return conflict("this report is %s and can no longer be edited", r.Status)
		}
		site, err := s.sites.GetByID(ctx, r.SiteID)
		if err != nil {
			return err
		}
		if problems := domain.ValidateIntervals(site, r.Date, in.Intervals, s.clock.Now(), s.loc); len(problems) > 0 {
			return invalidProblems(problems)
		}
		r.Notes, r.Intervals = in.Notes, in.Intervals
		saved, err = s.reports.SaveDraft(ctx, r, rowVersion)
		return err
	})
	return saved, err
}

// Submit hands a draft to the supervisor. Submitting something already
// submitted does nothing, so a retry after a dropped connection is safe.
func (s *ReportService) Submit(ctx context.Context, actor domain.Actor, id string) (domain.DailyReport, error) {
	if !actor.Can(domain.PermReportsCreate) {
		return domain.DailyReport{}, apperrors.ErrForbidden
	}
	var out domain.DailyReport
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if r.AuthorID != actor.UserID {
			return apperrors.ErrForbidden
		}
		if r.Status == domain.ReportSubmitted {
			out = r
			return nil
		}
		if !r.Editable() {
			return conflict("this report is %s and cannot be submitted", r.Status)
		}
		site, err := s.sites.GetByID(ctx, r.SiteID)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		problems := domain.ValidateIntervals(site, r.Date, r.Intervals, now, s.loc)
		if !hasObserved(r.Intervals) {
			problems = append(problems, "add at least one observed interval before submitting")
		}
		if len(problems) > 0 {
			return invalidProblems(problems)
		}
		if err := s.reports.MarkSubmitted(ctx, id, now); err != nil {
			return err
		}
		out, err = s.reports.GetByID(ctx, id, false)
		if err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "report.submitted", EntityType: "daily_report", EntityID: id,
			Metadata: map[string]any{"site_id": r.SiteID, "date": dateString(r.Date), "revision": r.Revision},
		})
	})
	return out, err
}

func hasObserved(ivs []domain.PowerInterval) bool {
	for _, iv := range ivs {
		if iv.Basis == domain.BasisObserved {
			return true
		}
	}
	return false
}

// Approve publishes a submitted report. The approver must be someone other
// than the author, and approving a correction supersedes the report it
// corrects in the same transaction.
func (s *ReportService) Approve(ctx context.Context, actor domain.Actor, id, comment string) (domain.DailyReport, error) {
	return s.review(ctx, actor, id, domain.ReportApproved, comment)
}

// Reject returns a submitted report to its author with a reason.
func (s *ReportService) Reject(ctx context.Context, actor domain.Actor, id, reason string) (domain.DailyReport, error) {
	if reason == "" {
		return domain.DailyReport{}, invalid("say why the report is being returned")
	}
	return s.review(ctx, actor, id, domain.ReportRejected, reason)
}

func (s *ReportService) review(ctx context.Context, actor domain.Actor, id string, outcome domain.ReportStatus, comment string) (domain.DailyReport, error) {
	if !actor.Can(domain.PermReportsReview) {
		return domain.DailyReport{}, apperrors.ErrForbidden
	}
	var out domain.DailyReport
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		r, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if r.Status != domain.ReportSubmitted {
			return conflict("only a submitted report can be reviewed; this one is %s", r.Status)
		}
		if r.AuthorID == actor.UserID {
			return forbidden("you cannot review your own entry")
		}
		site, err := s.sites.GetByID(ctx, r.SiteID)
		if err != nil {
			return err
		}

		meta := map[string]any{
			"site_id": r.SiteID, "date": dateString(r.Date), "revision": r.Revision, "outcome": string(outcome),
		}
		if outcome == domain.ReportApproved && r.SupersedesID != "" {
			prev, err := s.reports.GetByID(ctx, r.SupersedesID, true)
			if err != nil {
				return err
			}
			// The earlier revision steps aside first: only one report per
			// site and day may be approved at a time.
			if err := s.reports.MarkSuperseded(ctx, prev.ID); err != nil {
				return err
			}
			meta["supersedes_id"] = prev.ID
			meta["reason"] = r.CorrectionReason
			meta["before"] = summaryMeta(domain.Summarise(site, prev.Date, prev.Intervals, s.loc))
			meta["after"] = summaryMeta(domain.Summarise(site, r.Date, r.Intervals, s.loc))
		}
		now := s.clock.Now()
		if err := s.reports.MarkReviewed(ctx, id, outcome, actor.UserID, comment, now); err != nil {
			return err
		}
		if out, err = s.reports.GetByID(ctx, id, false); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "report." + string(outcome), EntityType: "daily_report", EntityID: id, Metadata: meta,
		})
	})
	return out, err
}

func summaryMeta(sum domain.DailySummary) map[string]any {
	return map[string]any{"served_minutes": sum.ServedMinutes, "outage_minutes": sum.OutageMinutes, "unknown_minutes": sum.UnknownMinutes}
}

// StartCorrection opens a new draft revision of an approved report. The
// original stays approved and untouched until the correction is approved.
func (s *ReportService) StartCorrection(ctx context.Context, actor domain.Actor, id, reason string) (domain.DailyReport, error) {
	if !actor.Can(domain.PermReportsCreate) {
		return domain.DailyReport{}, apperrors.ErrForbidden
	}
	if len(reason) < 5 {
		return domain.DailyReport{}, invalid("explain why the approved report needs correcting")
	}
	var out domain.DailyReport
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		orig, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if orig.Status != domain.ReportApproved {
			return conflict("only an approved report can be corrected; this one is %s", orig.Status)
		}
		rev, err := s.reports.NextRevision(ctx, orig.SiteID, orig.Date)
		if err != nil {
			return err
		}
		intervals := make([]domain.PowerInterval, len(orig.Intervals))
		for i, iv := range orig.Intervals {
			iv.ID = ""
			intervals[i] = iv
		}
		out, err = s.reports.Create(ctx, domain.DailyReport{
			SiteID: orig.SiteID, Date: orig.Date, Revision: rev, Status: domain.ReportDraft,
			AuthorID: actor.UserID, Notes: orig.Notes, Intervals: intervals,
			SupersedesID: orig.ID, CorrectionReason: reason,
		})
		if err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "report.correction_started", EntityType: "daily_report", EntityID: out.ID,
			Metadata: map[string]any{"supersedes_id": orig.ID, "reason": reason, "revision": rev},
		})
	})
	return out, err
}

// Get returns a report with its computed summary.
func (s *ReportService) Get(ctx context.Context, actor domain.Actor, id string) (ReportView, error) {
	if !actor.Can(domain.PermReportsRead) && !actor.Can(domain.PermPublishedRead) {
		return ReportView{}, apperrors.ErrForbidden
	}
	r, err := s.load(ctx, actor, id, false)
	if err != nil {
		return ReportView{}, err
	}
	site, err := s.sites.GetByID(ctx, r.SiteID)
	if err != nil {
		return ReportView{}, err
	}
	return s.view(r, site), nil
}

func (s *ReportService) view(r domain.DailyReport, site domain.Site) ReportView {
	dayStart, _ := domain.DayBounds(r.Date, s.loc)
	var warnings []string
	if n := countDeclared(r.Intervals); n > 0 {
		warnings = append(warnings, "declared intervals are shown but not counted: only observed intervals establish availability")
	}
	return ReportView{
		Report: r, Site: site,
		Summary: domain.Summarise(site, r.Date, r.Intervals, s.loc),
		Calculation: Calculation{
			Formula: formulaDailyAvailability, Version: formulaVersion,
			ReportID: r.ID, Revision: r.Revision,
			From:       dayStart.Add(time.Duration(site.ServiceStartMinute) * time.Minute),
			To:         dayStart.Add(time.Duration(site.ServiceEndMinute) * time.Minute),
			ComputedAt: s.clock.Now(), Warnings: warnings,
		},
	}
}

func countDeclared(ivs []domain.PowerInterval) int {
	n := 0
	for _, iv := range ivs {
		if iv.Basis == domain.BasisDeclared {
			n++
		}
	}
	return n
}

type ListReportsInput struct {
	SiteID        string
	From, To      *time.Time
	Statuses      []domain.ReportStatus
	Limit, Offset int
}

// List returns reports within the caller's scope. People who can only see
// published records get approved reports and nothing else.
func (s *ReportService) List(ctx context.Context, actor domain.Actor, in ListReportsInput) ([]domain.DailyReport, int, error) {
	canInternal := actor.Can(domain.PermReportsRead)
	if !canInternal && !actor.Can(domain.PermPublishedRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	if in.SiteID != "" && !actor.CanAccessSite(in.SiteID) {
		return nil, 0, apperrors.ErrNotFound
	}
	for _, st := range in.Statuses {
		if !domain.ValidReportStatus(st) {
			return nil, 0, invalid("%q is not a known report status", st)
		}
	}
	statuses := in.Statuses
	if !canInternal {
		statuses = []domain.ReportStatus{domain.ReportApproved}
	}
	limit, offset := Page(in.Limit, in.Offset)
	return s.reports.List(ctx, ports.ReportFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), SiteID: in.SiteID,
		From: in.From, To: in.To, Statuses: statuses, Limit: limit, Offset: offset,
	})
}

func dateString(t time.Time) string { return t.Format("2006-01-02") }
