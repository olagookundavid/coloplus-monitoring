package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// WithAcceptances lets the exports include the commissioning backlog.
func (s *ExportService) WithAcceptances(a ports.AcceptanceRepository) *ExportService {
	s.acceptances = a
	return s
}

// PerformanceRow is one site's month: the pooled figures and the facts the
// figures rest on.
type PerformanceRow struct {
	Site    domain.Site
	Summary domain.DailySummary
	Quality domain.DataQuality
	// GeneratorHours are the hours the approved reports show the generator ran.
	GeneratorHours   float64
	IncidentsStarted int
	// Fingerprint identifies the approved reports the figures came from.
	Fingerprint string
	// SLA judges the month's figure against the target; nil when none is set.
	SLA *domain.SLAResult
}

// PerformancePack is the monthly performance pack: one row per site.
type PerformancePack struct {
	From, To    time.Time
	Timezone    string
	GeneratedAt time.Time
	Formula     string
	Version     string
	Rows        []PerformanceRow
}

func (s *ExportService) today() time.Time {
	y, m, d := s.clock.Now().In(s.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// MonthlyPerformance builds the pack for the calendar month starting at month
// (the current month runs to today). Availability follows each site's own
// reading of unreported time, with the assumed minutes shown beside it.
func (s *ExportService) MonthlyPerformance(ctx context.Context, actor domain.Actor, month time.Time, region string) (PerformancePack, error) {
	out, err := s.performance(ctx, actor, month, region)
	if err != nil {
		return PerformancePack{}, err
	}
	if err := s.auditExport(ctx, actor, "monthly_performance", out.From, out.To, region, len(out.Rows)); err != nil {
		return PerformancePack{}, err
	}
	return out, nil
}

// performance works the month out without recording anything; the export and
// the stored run both build on it.
func (s *ExportService) performance(ctx context.Context, actor domain.Actor, month time.Time, region string) (PerformancePack, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return PerformancePack{}, apperrors.ErrForbidden
	}
	from := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)
	if today := s.today(); from.After(today) {
		return PerformancePack{}, invalid("that month has not started")
	} else if to.After(today) {
		to = today
	}
	days := int(to.Sub(from).Hours()/24) + 1

	sites, err := s.scopedSites(ctx, actor, "", region)
	if err != nil {
		return PerformancePack{}, err
	}
	if len(sites)*days > maxExportRows {
		return PerformancePack{}, invalid("that is too many rows; pick one region")
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}
	reps, err := s.reports.ApprovedInRange(ctx, ids, from, to)
	if err != nil {
		return PerformancePack{}, err
	}
	bySite := map[string][]domain.DailyReport{}
	byDay := map[string]domain.DailyReport{}
	for _, r := range reps {
		bySite[r.SiteID] = append(bySite[r.SiteID], r)
		byDay[ports.DayKey(r.SiteID, r.Date)] = r
	}
	started, err := s.incidentsStarted(ctx, actor, ids, from, to)
	if err != nil {
		return PerformancePack{}, err
	}
	adj, err := s.adjust.For(ctx, ids, from, to, byDay)
	if err != nil {
		return PerformancePack{}, err
	}

	out := PerformancePack{
		From: from, To: to, Timezone: s.loc.String(), GeneratedAt: s.clock.Now(),
		Formula: formulaDailyAvailability, Version: formulaVersion, Rows: make([]PerformanceRow, 0, len(sites)),
	}
	for _, site := range sites {
		var sums []domain.DailySummary
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			if r, ok := byDay[ports.DayKey(site.ID, d)]; ok {
				sums = append(sums, adj.summarise(s.loc, site, d, r.Intervals, true))
			} else {
				sums = append(sums, adj.summarise(s.loc, site, d, nil, false))
			}
		}
		pooled := domain.PortfolioSummary(sums)
		row := PerformanceRow{
			Site: site, Summary: pooled, IncidentsStarted: started[site.ID],
			Quality: domain.AssessQuality(days, bySite[site.ID], pooled),
		}
		ids := make([]string, 0, len(bySite[site.ID]))
		for _, r := range bySite[site.ID] {
			row.GeneratorHours += domain.GeneratorHoursOf(r)
			ids = append(ids, r.ID)
		}
		sort.Strings(ids)
		sum := sha256.Sum256([]byte(strings.Join(ids, ",")))
		row.Fingerprint = hex.EncodeToString(sum[:])
		row.SLA = s.sla.Assess(pooled)
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// incidentsStarted counts, per site, the incidents that started in the period;
// a customer counts only the published ones.
func (s *ExportService) incidentsStarted(ctx context.Context, actor domain.Actor, siteIDs []string, from, to time.Time) (map[string]int, error) {
	out := map[string]int{}
	if !actor.Can(domain.PermIncidentsRead) && !actor.Can(domain.PermPublishedRead) {
		return out, nil
	}
	start, _ := domain.DayBounds(from, s.loc)
	_, end := domain.DayBounds(to, s.loc)
	list, _, err := s.incidents.List(ctx, ports.IncidentFilter{
		SiteIDs: siteIDs, PublishedOnly: !actor.Can(domain.PermIncidentsRead), StartedFrom: &start, StartedBefore: &end, Limit: maxExportRows,
	})
	if err != nil {
		return nil, err
	}
	for _, i := range list {
		out[i.SiteID]++
	}
	return out, nil
}

// ExceptionRow is a site with something to look at in the week.
type ExceptionRow struct {
	Site              domain.Site
	OutageMinutes     int
	OutageEvents      int
	DaysWithoutReport int
	LateReports       int
	Corrections       int
	// The open counts are as of the time the report is made.
	OpenIncidents    int
	OverdueIncidents int
	OpenDisputes     int
	// Reasons names why the site is listed.
	Reasons []string
}

type ExceptionReport struct {
	From, To    time.Time
	Timezone    string
	GeneratedAt time.Time
	SitesInView int
	Rows        []ExceptionRow
}

// WeeklyExceptions lists, for the week starting on the given Monday, only the
// sites that had an outage, a day with no report, a late report, a correction,
// or an open incident or dispute. A site with nothing to report is left out and
// counted in SitesInView, so a short list is not mistaken for a short portfolio.
func (s *ExportService) WeeklyExceptions(ctx context.Context, actor domain.Actor, weekStart time.Time, region string) (ExceptionReport, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return ExceptionReport{}, apperrors.ErrForbidden
	}
	if weekStart.Weekday() != time.Monday {
		return ExceptionReport{}, invalid("a week starts on a Monday")
	}
	from, to := weekStart, weekStart.AddDate(0, 0, 6)
	if today := s.today(); from.After(today) {
		return ExceptionReport{}, invalid("that week has not started")
	} else if to.After(today) {
		to = today
	}
	days := int(to.Sub(from).Hours()/24) + 1

	sites, err := s.scopedSites(ctx, actor, "", region)
	if err != nil {
		return ExceptionReport{}, err
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}
	reps, err := s.reports.ApprovedInRange(ctx, ids, from, to)
	if err != nil {
		return ExceptionReport{}, err
	}
	bySite := map[string][]domain.DailyReport{}
	for _, r := range reps {
		bySite[r.SiteID] = append(bySite[r.SiteID], r)
	}

	now := s.clock.Now()
	staffIncidents := actor.Can(domain.PermIncidentsRead)
	open := map[string]int{}
	overdue := map[string]int{}
	if staffIncidents || actor.Can(domain.PermPublishedRead) {
		list, _, err := s.incidents.List(ctx, ports.IncidentFilter{SiteIDs: ids, PublishedOnly: !staffIncidents, OpenOnly: true, Limit: maxExportRows})
		if err != nil {
			return ExceptionReport{}, err
		}
		for _, i := range list {
			open[i.SiteID]++
			if i.Overdue(now) {
				overdue[i.SiteID]++
			}
		}
	}
	disputes := map[string]int{}
	if actor.Can(domain.PermDisputesRead) {
		list, _, err := s.disputes.List(ctx, ports.DisputeFilter{SiteIDs: ids, OpenOnly: true, Limit: maxExportRows})
		if err != nil {
			return ExceptionReport{}, err
		}
		for _, d := range list {
			disputes[d.SiteID]++
		}
	}

	out := ExceptionReport{From: from, To: to, Timezone: s.loc.String(), GeneratedAt: now, SitesInView: len(sites)}
	for _, site := range sites {
		row := ExceptionRow{
			Site: site, DaysWithoutReport: days - len(bySite[site.ID]),
			OpenIncidents: open[site.ID], OverdueIncidents: overdue[site.ID], OpenDisputes: disputes[site.ID],
		}
		for _, r := range bySite[site.ID] {
			row.OutageMinutes += domain.Summarise(site, r.Date, r.Intervals, s.loc).OutageMinutes
			row.OutageEvents += len(domain.OutageSpans(site, r.Date, r.Intervals, s.loc))
			if r.LateReason != "" {
				row.LateReports++
			}
			if r.SupersedesID != "" {
				row.Corrections++
			}
		}
		add := func(cond bool, why string) {
			if cond {
				row.Reasons = append(row.Reasons, why)
			}
		}
		add(row.OutageMinutes > 0, "outage")
		add(row.DaysWithoutReport > 0, "days_without_report")
		add(row.LateReports > 0, "late_report")
		add(row.Corrections > 0, "correction")
		add(row.OpenIncidents > 0, "open_incident")
		add(row.OverdueIncidents > 0, "overdue_incident")
		add(row.OpenDisputes > 0, "open_dispute")
		if len(row.Reasons) > 0 {
			out.Rows = append(out.Rows, row)
		}
	}
	sort.SliceStable(out.Rows, func(i, j int) bool { return len(out.Rows[i].Reasons) > len(out.Rows[j].Reasons) })
	if err := s.auditExport(ctx, actor, "weekly_exceptions", from, to, region, len(out.Rows)); err != nil {
		return ExceptionReport{}, err
	}
	return out, nil
}

// BacklogRow is one site's commissioning, from the inspection request to acceptance.
type BacklogRow struct {
	Site       domain.Site
	Acceptance domain.Acceptance
	Overdue    bool
	// ReadyAgeHours is how long the site has waited since it was ready for
	// inspection; it is reported apart from operational performance.
	ReadyAgeHours int
}

type CommissioningBacklog struct {
	GeneratedAt time.Time
	Timezone    string
	Rows        []BacklogRow
}

// CommissioningBacklog lists the sites that are not yet accepted, longest
// waiting first.
func (s *ExportService) CommissioningBacklog(ctx context.Context, actor domain.Actor, region string) (CommissioningBacklog, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return CommissioningBacklog{}, apperrors.ErrForbidden
	}
	if !actor.Can(domain.PermAcceptanceRead) || s.acceptances == nil {
		return CommissioningBacklog{}, apperrors.ErrForbidden
	}
	sites, err := s.scopedSites(ctx, actor, "", region)
	if err != nil {
		return CommissioningBacklog{}, err
	}
	bySite := make(map[string]domain.Site, len(sites))
	ids := make([]string, len(sites))
	for i, site := range sites {
		bySite[site.ID] = site
		ids[i] = site.ID
	}
	list, _, err := s.acceptances.List(ctx, ports.AcceptanceFilter{SiteIDs: ids, OpenOnly: true, Limit: maxExportRows})
	if err != nil {
		return CommissioningBacklog{}, err
	}
	now := s.clock.Now()
	out := CommissioningBacklog{GeneratedAt: now, Timezone: s.loc.String()}
	for _, a := range list {
		site, ok := bySite[a.SiteID]
		if !ok {
			continue
		}
		out.Rows = append(out.Rows, BacklogRow{
			Site: site, Acceptance: a, Overdue: a.Overdue(now), ReadyAgeHours: int(a.ReadyAge(now).Hours()),
		})
	}
	if err := s.auditExport(ctx, actor, "commissioning_backlog", now, now, region, len(out.Rows)); err != nil {
		return CommissioningBacklog{}, err
	}
	return out, nil
}

func (s *ExportService) auditExport(ctx context.Context, actor domain.Actor, name string, from, to time.Time, region string, rows int) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		meta := map[string]any{"export": name, "from": dateString(from), "to": dateString(to), "rows": rows}
		if region != "" {
			meta["region"] = region
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "export." + name, EntityType: "export",
			EntityID: fmt.Sprintf("%s:%s:%s", name, dateString(from), dateString(to)), Metadata: meta,
		})
	})
}
