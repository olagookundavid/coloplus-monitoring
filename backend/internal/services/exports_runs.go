package services

import (
	"context"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

const runKindMonthly = "monthly_performance"

// WithAdjustments makes the exports take planned maintenance and the midnight
// carry-over into account.
func (s *ExportService) WithAdjustments(a *Adjustments) *ExportService {
	s.adjust = a
	return s
}

// WithSLA sets the availability target the monthly pack judges sites against.
func (s *ExportService) WithSLA(sla domain.SLA) *ExportService {
	s.sla = sla
	return s
}

// WithRuns lets the service store calculation runs.
func (s *ExportService) WithRuns(r ports.CalculationRunRepository) *ExportService {
	s.runs = r
	return s
}

// runsAllowed: a run is the portfolio's statement, kept for Coloplus. It is for
// exporters who see every site and are not the customer.
func runsAllowed(actor domain.Actor) bool {
	return actor.Can(domain.PermExportsCreate) && actor.AllSites && !actor.Can(domain.PermPublishedRead)
}

// StoreRun works out a month for every site (optionally one region) and freezes
// the result with the formula version and a fingerprint of the approved reports
// it rests on. It changes nothing else. Storing it again later makes a second,
// separate run.
func (s *ExportService) StoreRun(ctx context.Context, actor domain.Actor, month time.Time, region, label string) (domain.CalculationRun, error) {
	if !runsAllowed(actor) {
		return domain.CalculationRun{}, apperrors.ErrForbidden
	}
	label = strings.TrimSpace(label)
	if len(label) > 120 {
		return domain.CalculationRun{}, invalid("the label is at most 120 characters")
	}
	pack, err := s.performance(ctx, actor, month, strings.TrimSpace(region))
	if err != nil {
		return domain.CalculationRun{}, err
	}
	run := domain.CalculationRun{
		Kind: runKindMonthly, PeriodFrom: pack.From, PeriodTo: pack.To, Region: strings.TrimSpace(region), Label: label,
		Formula: pack.Formula, FormulaVersion: pack.Version, CreatedBy: actor.UserID, CreatedAt: s.clock.Now(),
		Rows: make([]domain.RunRow, len(pack.Rows)),
	}
	for i, r := range pack.Rows {
		run.Rows[i] = runRowOf(r, pack.From, pack.To)
	}
	var stored domain.CalculationRun
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if stored, err = s.runs.Create(ctx, run); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "calculation_run.stored", EntityType: "calculation_run", EntityID: stored.ID,
			Metadata: map[string]any{"from": dateString(pack.From), "to": dateString(pack.To), "region": run.Region, "sites": len(run.Rows), "formula_version": pack.Version},
		})
	})
	return stored, err
}

func runRowOf(r PerformanceRow, from, to time.Time) domain.RunRow {
	return domain.RunRow{
		SiteID: r.Site.ID, CustomerSiteCode: r.Site.CustomerSiteCode, SiteName: r.Site.Name,
		EligibleMinutes: r.Summary.EligibleMinutes, ServedMinutes: r.Summary.ServedMinutes, OutageMinutes: r.Summary.OutageMinutes,
		UnknownMinutes: r.Summary.UnknownMinutes, AssumedMinutes: r.Summary.AssumedMinutes,
		DaysInPeriod: r.Quality.Days, DaysReported: r.Quality.ReportedDays,
		ReportedAvailability: r.Summary.ReportedAvailability, ConfirmedServedShare: r.Summary.ConfirmedServedShare, Coverage: r.Summary.Coverage,
		GeneratorHours: r.GeneratorHours, IncidentsStarted: r.IncidentsStarted, Fingerprint: r.Fingerprint,
	}
}

// Runs lists stored runs, newest first.
func (s *ExportService) Runs(ctx context.Context, actor domain.Actor, limit, offset int) ([]domain.CalculationRun, int, error) {
	if !runsAllowed(actor) {
		return nil, 0, apperrors.ErrForbidden
	}
	limit, offset = Page(limit, offset)
	return s.runs.List(ctx, limit, offset)
}

// Run returns one stored run with its rows.
func (s *ExportService) Run(ctx context.Context, actor domain.Actor, id string) (domain.CalculationRun, error) {
	if !runsAllowed(actor) {
		return domain.CalculationRun{}, apperrors.ErrForbidden
	}
	return s.runs.Get(ctx, id)
}

// RunChange is one site whose figures differ between a stored run and today.
type RunChange struct {
	SiteID, CustomerSiteCode, SiteName string
	// Then is nil for a site that was not in the run; Now is nil for a site
	// that no longer appears.
	Then, Now *domain.RunRow
	// ReportsChanged says the approved reports behind the figures are not the
	// same ones any more (a correction, or a report approved after the run).
	ReportsChanged bool
}

// RunComparison is a stored run set against the same period worked out today.
type RunComparison struct {
	Run         domain.CalculationRun
	GeneratedAt time.Time
	// FormulaChanged is true when today's formula version is not the run's;
	// then differences may come from the formula rather than from the reports.
	FormulaChanged bool
	CurrentVersion string
	Unchanged      int
	Changes        []RunChange
}

// CompareRun sets a stored run against the same month and region worked out
// now, listing only the sites whose figures differ. The run itself is never
// touched.
func (s *ExportService) CompareRun(ctx context.Context, actor domain.Actor, id string) (RunComparison, error) {
	if !runsAllowed(actor) {
		return RunComparison{}, apperrors.ErrForbidden
	}
	run, err := s.runs.Get(ctx, id)
	if err != nil {
		return RunComparison{}, err
	}
	pack, err := s.performance(ctx, actor, run.PeriodFrom, run.Region)
	if err != nil {
		return RunComparison{}, err
	}
	// A month still running at the time has grown since; the run's own end day
	// is the end of what it states, so compare like with like.
	if !pack.To.Equal(run.PeriodTo) {
		return RunComparison{}, invalid("this run covers %s to %s; its month has moved on, so it cannot be compared day for day", dateString(run.PeriodFrom), dateString(run.PeriodTo))
	}

	out := RunComparison{Run: run, GeneratedAt: s.clock.Now(), CurrentVersion: pack.Version, FormulaChanged: pack.Version != run.FormulaVersion}
	then := map[string]domain.RunRow{}
	for _, r := range run.Rows {
		then[r.SiteID] = r
	}
	for _, pr := range pack.Rows {
		now := runRowOf(pr, pack.From, pack.To)
		t, ok := then[now.SiteID]
		delete(then, now.SiteID)
		switch {
		case !ok:
			out.Changes = append(out.Changes, RunChange{SiteID: now.SiteID, CustomerSiteCode: now.CustomerSiteCode, SiteName: now.SiteName, Now: &now, ReportsChanged: true})
		case t.SameFigures(now) && t.Fingerprint == now.Fingerprint:
			out.Unchanged++
		default:
			tt := t
			out.Changes = append(out.Changes, RunChange{
				SiteID: now.SiteID, CustomerSiteCode: now.CustomerSiteCode, SiteName: now.SiteName, Then: &tt, Now: &now,
				ReportsChanged: t.Fingerprint != now.Fingerprint,
			})
		}
	}
	for _, t := range then {
		tt := t
		out.Changes = append(out.Changes, RunChange{SiteID: t.SiteID, CustomerSiteCode: t.CustomerSiteCode, SiteName: t.SiteName, Then: &tt, ReportsChanged: true})
	}
	return out, nil
}
