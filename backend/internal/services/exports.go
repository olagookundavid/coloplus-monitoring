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

// ExportService produces figures to take out of the system. Every row says
// where it came from: the report revision, the basis, the coverage, the
// formula and its version, the dispute state and when it was generated. An
// export is audited in the same transaction that decided it was allowed.
type ExportService struct {
	sites     ports.SiteRepository
	reports   ports.ReportRepository
	disputes  ports.DisputeRepository
	incidents ports.IncidentRepository
	tx        ports.TxManager
	audit     *AuditService
	clock     ports.Clock
	loc       *time.Location
}

func NewExportService(
	sites ports.SiteRepository, reports ports.ReportRepository, disputes ports.DisputeRepository,
	incidents ports.IncidentRepository, tx ports.TxManager, audit *AuditService, clock ports.Clock, loc *time.Location,
) *ExportService {
	return &ExportService{sites: sites, reports: reports, disputes: disputes, incidents: incidents, tx: tx, audit: audit, clock: clock, loc: loc}
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
	Region   string
	From, To time.Time
}

// checkRange validates the first and last day of an export.
func (s *ExportService) checkRange(from, to time.Time) (days int, err error) {
	y, m, d := s.clock.Now().In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch {
	case from.IsZero() || to.IsZero():
		return 0, invalid("give the first and last day to export")
	case to.Before(from):
		return 0, invalid("the last day is before the first")
	case to.After(today):
		return 0, invalid("the last day cannot be in the future")
	}
	days = int(to.Sub(from).Hours()/24) + 1
	if days > maxExportDays {
		return 0, invalid("export at most %d days at a time", maxExportDays)
	}
	return days, nil
}

// scopedSites are the caller's sites, narrowed to one site and/or region.
func (s *ExportService) scopedSites(ctx context.Context, actor domain.Actor, siteID, region string) ([]domain.Site, error) {
	if siteID != "" && !actor.CanAccessSite(siteID) {
		return nil, apperrors.ErrNotFound
	}
	all, _, err := s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Limit: overviewSiteLimit,
	})
	if err != nil {
		return nil, err
	}
	kept := all[:0]
	for _, site := range all {
		if (siteID == "" || site.ID == siteID) && (region == "" || strings.EqualFold(site.Region, region)) {
			kept = append(kept, site)
		}
	}
	return kept, nil
}

// DailySummaries exports one row per site and reporting day in the range,
// within the caller's scope. Days without an approved report are rows too.
func (s *ExportService) DailySummaries(ctx context.Context, actor domain.Actor, in ExportDailySummariesInput) (DailySummaryExport, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return DailySummaryExport{}, apperrors.ErrForbidden
	}
	days, err := s.checkRange(in.From, in.To)
	if err != nil {
		return DailySummaryExport{}, err
	}
	now := s.clock.Now()
	sites, err := s.scopedSites(ctx, actor, in.SiteID, in.Region)
	if err != nil {
		return DailySummaryExport{}, err
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
		if in.Region != "" {
			meta["region"] = in.Region
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

// IncidentRow is one reported fault in the register, with its site.
type IncidentRow struct {
	Incident domain.Incident
	Site     domain.Site
}

// IncidentRegisterExport is the fault register for a period.
type IncidentRegisterExport struct {
	From, To    time.Time
	Timezone    string
	GeneratedAt time.Time
	Rows        []IncidentRow
	// StaffView is true when unpublished incidents are included.
	StaffView bool
}

type ExportIncidentsInput struct {
	SiteID   string
	Region   string
	From, To time.Time
}

// ExportIncidentRegisterAssumptions travels with every incident row.
const ExportIncidentRegisterAssumptions = "Start and restoration are times recorded by Coloplus; " +
	"a restoration target is a plan and never counts as restoration. " +
	"Duration is blank until restoration is recorded. " +
	"An incident belongs to the period in which it started. " +
	"Faults with no outage are listed apart from service interruptions by their impact."

// IncidentRegister lists every incident that started in the range, within the
// caller's sites. Airtel gets published incidents only, as everywhere else.
func (s *ExportService) IncidentRegister(ctx context.Context, actor domain.Actor, in ExportIncidentsInput) (IncidentRegisterExport, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return IncidentRegisterExport{}, apperrors.ErrForbidden
	}
	if !actor.Can(domain.PermIncidentsRead) && !actor.Can(domain.PermPublishedRead) {
		return IncidentRegisterExport{}, apperrors.ErrForbidden
	}
	if _, err := s.checkRange(in.From, in.To); err != nil {
		return IncidentRegisterExport{}, err
	}
	sites, err := s.scopedSites(ctx, actor, in.SiteID, in.Region)
	if err != nil {
		return IncidentRegisterExport{}, err
	}
	bySite := make(map[string]domain.Site, len(sites))
	ids := make([]string, len(sites))
	for i, site := range sites {
		bySite[site.ID] = site
		ids[i] = site.ID
	}
	// A local day starts at local midnight, not at midnight UTC.
	dayStart, _ := domain.DayBounds(in.From, s.loc)
	_, dayEnd := domain.DayBounds(in.To, s.loc)
	staff := actor.Can(domain.PermIncidentsRead)
	list, total, err := s.incidents.List(ctx, ports.IncidentFilter{
		SiteIDs: ids, PublishedOnly: !staff, StartedFrom: &dayStart, StartedBefore: &dayEnd, Limit: maxExportRows,
	})
	if err != nil {
		return IncidentRegisterExport{}, err
	}
	if total > len(list) {
		return IncidentRegisterExport{}, invalid("that is too many incidents; pick fewer days or one site")
	}

	out := IncidentRegisterExport{
		From: in.From, To: in.To, Timezone: s.loc.String(), GeneratedAt: s.clock.Now(), StaffView: staff,
		Rows: make([]IncidentRow, 0, len(list)),
	}
	for _, i := range list {
		out.Rows = append(out.Rows, IncidentRow{Incident: i, Site: bySite[i.SiteID]})
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		meta := map[string]any{"export": "incident_register", "from": dateString(in.From), "to": dateString(in.To), "rows": len(out.Rows)}
		if in.SiteID != "" {
			meta["site_id"] = in.SiteID
		}
		if in.Region != "" {
			meta["region"] = in.Region
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "export.incident_register", EntityType: "export",
			EntityID: fmt.Sprintf("incident_register:%s:%s", dateString(in.From), dateString(in.To)), Metadata: meta,
		})
	})
	if err != nil {
		return IncidentRegisterExport{}, err
	}
	return out, nil
}
