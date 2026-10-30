package services

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// ExportService produces figures to take out of the system. Every row says
// where it came from: the report revision, the basis, the coverage, the
// formula and its version, the dispute state and when it was generated. An
// export is audited in the same transaction that decided it was allowed.
type ExportService struct {
	sites    ports.SiteRepository
	reports  ports.ReportRepository
	disputes ports.DisputeRepository
	tx       ports.TxManager
	audit    *AuditService
	clock    ports.Clock
	loc      *time.Location
}

func NewExportService(
	sites ports.SiteRepository, reports ports.ReportRepository, disputes ports.DisputeRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, loc *time.Location,
) *ExportService {
	return &ExportService{sites: sites, reports: reports, disputes: disputes, tx: tx, audit: audit, clock: clock, loc: loc}
}

const (
	maxExportDays = 92
	maxExportRows = 100_000

	// ExportAssumptions travels with every row, so a figure copied into a
	// spreadsheet still says what it does and does not mean.
	ExportAssumptions = "Observed intervals only; declared intervals are kept but never counted. " +
		"Time outside the service window is not counted. " +
		"Unknown time is neither served nor outage. " +
		"A day without an approved report has no figures and is not availability."
)

// DailySummaryRow is one site and one reporting day. Summary is nil when the
// day has no approved report: the row stays, with no figures.
type DailySummaryRow struct {
	Site    domain.Site
	Date    time.Time
	Report  *domain.DailyReport
	Summary domain.DailySummary
	// DeclaredIgnored counts declared intervals kept but not counted.
	DeclaredIgnored int
	Disputes        ports.DisputeDay
}

// DailySummaryExport is the data behind the CSV and everything needed to
// describe it.
type DailySummaryExport struct {
	From, To    time.Time
	Timezone    string
	GeneratedAt time.Time
	Formula     string
	Version     string
	Rows        []DailySummaryRow
}

type ExportDailySummariesInput struct {
	SiteID   string
	From, To time.Time
}

// DailySummaries exports one row per site and reporting day in the range,
// within the caller's scope. Days without an approved report are rows too.
func (s *ExportService) DailySummaries(ctx context.Context, actor domain.Actor, in ExportDailySummariesInput) (DailySummaryExport, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return DailySummaryExport{}, apperrors.ErrForbidden
	}
	if in.SiteID != "" && !actor.CanAccessSite(in.SiteID) {
		return DailySummaryExport{}, apperrors.ErrNotFound
	}
	now := s.clock.Now()
	y, m, d := now.In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch {
	case in.From.IsZero() || in.To.IsZero():
		return DailySummaryExport{}, invalid("give the first and last day to export")
	case in.To.Before(in.From):
		return DailySummaryExport{}, invalid("the last day is before the first")
	case in.To.After(today):
		return DailySummaryExport{}, invalid("the last day cannot be in the future")
	case int(in.To.Sub(in.From).Hours()/24)+1 > maxExportDays:
		return DailySummaryExport{}, invalid("export at most %d days at a time", maxExportDays)
	}
	days := int(in.To.Sub(in.From).Hours()/24) + 1

	sites, _, err := s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Limit: overviewSiteLimit,
	})
	if err != nil {
		return DailySummaryExport{}, err
	}
	if in.SiteID != "" {
		kept := sites[:0]
		for _, site := range sites {
			if site.ID == in.SiteID {
				kept = append(kept, site)
			}
		}
		sites = kept
	}
	if len(sites)*days > maxExportRows {
		return DailySummaryExport{}, invalid("that is too many rows; pick fewer days or one site")
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}

	reps, err := s.reports.ApprovedInRange(ctx, ids, in.From, in.To)
	if err != nil {
		return DailySummaryExport{}, err
	}
	byDay := make(map[string]domain.DailyReport, len(reps))
	for _, r := range reps {
		byDay[ports.DayKey(r.SiteID, r.Date)] = r
	}
	disputed, err := s.disputes.ByDay(ctx, ids, in.From, in.To)
	if err != nil {
		return DailySummaryExport{}, err
	}

	out := DailySummaryExport{
		From: in.From, To: in.To, Timezone: s.loc.String(), GeneratedAt: now,
		Formula: formulaDailyAvailability, Version: formulaVersion,
		Rows: make([]DailySummaryRow, 0, len(sites)*days),
	}
	for _, site := range sites {
		for date := in.From; !date.After(in.To); date = date.AddDate(0, 0, 1) {
			key := ports.DayKey(site.ID, date)
			row := DailySummaryRow{Site: site, Date: date, Disputes: disputed[key]}
			if r, ok := byDay[key]; ok {
				rep := r
				row.Report = &rep
				row.Summary = domain.Summarise(site, date, r.Intervals, s.loc)
				row.DeclaredIgnored = countDeclared(r.Intervals)
			} else {
				// No approved report: the whole window is unknown.
				row.Summary = domain.Summarise(site, date, nil, s.loc)
			}
			out.Rows = append(out.Rows, row)
		}
	}

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		meta := map[string]any{
			"export": "daily_summaries", "from": dateString(in.From), "to": dateString(in.To),
			"sites": len(sites), "rows": len(out.Rows),
		}
		if in.SiteID != "" {
			meta["site_id"] = in.SiteID
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "export.daily_summaries", EntityType: "export",
			EntityID: fmt.Sprintf("daily_summaries:%s:%s", dateString(in.From), dateString(in.To)), Metadata: meta,
		})
	})
	if err != nil {
		return DailySummaryExport{}, err
	}
	return out, nil
}
