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
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/ports"
)

// DashboardService builds the portfolio view from approved reports only.
type DashboardService struct {
	sites     ports.SiteRepository
	reports   ports.ReportRepository
	incidents ports.IncidentRepository
	fuel      ports.FuelRepository
	// fuelLowHours is the run time left at or under which a tank counts as low.
	fuelLowHours float64
	// sla is the availability target; the zero value judges nothing.
	sla domain.SLA
	// adjust adds planned maintenance and the midnight carry-over to the figures.
	adjust *Adjustments
	clock  ports.Clock
	loc    *time.Location

	// Identical questions asked at the same moment are computed once, and an
	// answer is kept for a few seconds. The key is the caller's exact site
	// scope, the period and the region, so one scope never sees another's.
	memo *platform.Memo
}

const (
	dashboardCacheMax       = 256
	dashboardComputeTimeout = 20 * time.Second
)

// NewDashboardService builds the portfolio view. cacheTTL is how long an
// answer is reused; it must stay well under the 60-second target for an
// approved entry to show on the dashboard.
func NewDashboardService(
	sites ports.SiteRepository, reports ports.ReportRepository, incidents ports.IncidentRepository,
	fuel ports.FuelRepository, clock ports.Clock, loc *time.Location, cacheTTL time.Duration, fuelLowHours float64,
) *DashboardService {
	return &DashboardService{
		sites: sites, reports: reports, incidents: incidents, fuel: fuel, fuelLowHours: fuelLowHours, clock: clock, loc: loc,
		memo: platform.NewMemo(clock, cacheTTL, dashboardComputeTimeout, dashboardCacheMax),
	}
}

// scopeKey identifies exactly which sites a caller may see.
func scopeKey(a domain.Actor) string {
	if a.AllSites {
		return "all"
	}
	ids := a.SiteIDList()
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, ",")))
	return "sites:" + hex.EncodeToString(sum[:16])
}

func (p Period) key() string {
	return fmt.Sprintf("%s|%s|%s", p.From.Format("2006-01-02"), p.To.Format("2006-01-02"), strings.ToLower(p.Region))
}

// SiteRow is one site's line on the dashboard.
type SiteRow struct {
	Site      domain.Site
	Condition domain.SiteCondition
	// AsOf is the end of the latest observed interval; nil when none.
	AsOf *time.Time
	// CurrentSource is what the site was last observed running on. Empty when
	// the report is stale, the last interval was an outage or none was observed.
	CurrentSource domain.PowerSource
	// Summary pools the period's days (a day with no approved report counts
	// its window as unknown). Nil when no day in the period has an approved
	// report, which is never shown as zero or as available.
	Summary *domain.DailySummary
	// DaysReported is how many days of the period have an approved report;
	// DaysWithoutReport the rest, shown beside the figure so an assumed day is never hidden.
	DaysReported      int
	DaysWithoutReport int
	// Fuel is the estimated tank level for the site; nil when the caller may
	// not see fuel, or the site has no agreed rate or nothing recorded.
	Fuel *FuelBrief
	// Quality says how much of the figure rests on reports.
	Quality domain.DataQuality
	// SLA judges the site's figure against the target; nil when none is set.
	SLA *domain.SLAResult
	// TodayReport/YesterdayReport are the report that best represents the
	// period's last day and the day before (in-progress if there is one, else
	// approved); nil when nothing has ever been filed. This is the reporter's
	// own "did I report today" view and must not be confused with Summary
	// above: a submitted-but-not-yet-approved report shows here but not there.
	TodayReport, YesterdayReport *domain.DailyReport
}

// FuelBrief is a site's fuel on one line. It is an estimate; see domain.FuelEstimate.
type FuelBrief struct {
	Litres         *float64
	HoursRemaining *float64
	Low            bool
	Empty          bool
	// Provisional means days since the last reading have no approved hours, so
	// the level is the most the tank could hold.
	Provisional bool
}

type StatusCounts struct {
	Active, PendingAcceptance, Suspended int
}

// WithAdjustments makes the figures take planned maintenance and the midnight
// carry-over into account.
func (s *DashboardService) WithAdjustments(a *Adjustments) *DashboardService {
	s.adjust = a
	return s
}

// WithSLA sets the availability target the dashboard judges figures against.
func (s *DashboardService) WithSLA(sla domain.SLA) *DashboardService {
	s.sla = sla
	return s
}

// Period is an inclusive range of reporting days, optionally narrowed to one
// region.
type Period struct {
	From, To time.Time
	Region   string
}

// Days is the number of reporting days in the period.
func (p Period) Days() int { return int(p.To.Sub(p.From).Hours()/24) + 1 }

type Overview struct {
	Period   Period
	Timezone string
	// Regions lists every region in the caller's scope, whatever the filter,
	// so the filter can offer them.
	Regions []string
	Counts  StatusCounts
	// Portfolio pools active sites by eligible minutes. A site with no
	// approved report counts its whole window as unknown.
	Portfolio           domain.DailySummary
	SitesReported       int
	SitesNotReported    int
	SitesAwaitingUpdate int
	SitesOutageReported int
	// SiteDaysWithoutReport counts, over the active sites, the days in the period
	// with no approved report.
	SiteDaysWithoutReport int
	// Quality is the indicator for the active sites together.
	Quality domain.DataQuality
	// SLA judges the portfolio's figure against the target; nil when none is set.
	SLA *domain.SLAResult
	// OutagesThisWeek counts distinct site/day pairs in the 7 days ending on
	// the period's last day whose current report has an outage. An
	// approximation, not a full weekly aggregation.
	OutagesThisWeek int
	Sites           []SiteRow
	GeneratedAt     time.Time
}

// TrendPoint is the portfolio on one reporting day.
type TrendPoint struct {
	// Date is the first day of the bucket (the day itself, the Monday of the
	// week, or the first of the month); From and To are the days of the bucket
	// that fall inside the requested period, so a part-bucket is plain.
	Date     time.Time
	From, To time.Time
	Days     int
	// ActiveSites is how many active sites were expected to report;
	// SitesReported how many had an approved report on at least one day of the
	// bucket.
	ActiveSites, SitesReported int
	// SiteDays are the active sites times the days in the bucket, and
	// SiteDaysReported how many of those have an approved report.
	SiteDays, SiteDaysReported int
	Summary                    domain.DailySummary
}

// Grain is the width of a trend point.
type Grain string

const (
	GrainDay   Grain = "day"
	GrainWeek  Grain = "week"
	GrainMonth Grain = "month"
)

// ParseGrain reads day (the default), week or month.
func ParseGrain(v string) (Grain, error) {
	switch Grain(v) {
	case "", GrainDay:
		return GrainDay, nil
	case GrainWeek, GrainMonth:
		return Grain(v), nil
	}
	return "", invalid("grain must be day, week or month")
}

// bucketStart is the first day of the bucket a date falls in. Weeks start on Monday.
func bucketStart(date time.Time, g Grain) time.Time {
	switch g {
	case GrainWeek:
		return date.AddDate(0, 0, -((int(date.Weekday()) + 6) % 7))
	case GrainMonth:
		return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return date
}

type Trend struct {
	// Grain is the width of each point: day, week or month.
	Grain       Grain
	Period      Period
	Timezone    string
	Formula     string
	Version     string
	GeneratedAt time.Time
	Points      []TrendPoint
}

const overviewSiteLimit = 1000

// scope is the sites and approved reports a dashboard question is about.
type scope struct {
	sites   []domain.Site
	regions []string
	byDay   map[string]domain.DailyReport
	adj     *dayAdjuster
	now     time.Time
}

// SinglePeriod is the one reporting day d.
func SinglePeriod(d time.Time) Period { return Period{From: d, To: d} }

func (s *DashboardService) validate(p Period, siteCount int) error {
	now := s.clock.Now()
	y, m, d := now.In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch {
	case p.From.IsZero() || p.To.IsZero():
		return invalid("give the first and last day")
	case p.To.Before(p.From):
		return invalid("the last day is before the first")
	case p.To.After(today):
		return invalid("the last day cannot be in the future")
	case p.Days() > maxExportDays:
		return invalid("choose at most %d days", maxExportDays)
	case siteCount*p.Days() > maxExportRows:
		return invalid("that is too many sites and days; choose fewer days or one region")
	}
	return nil
}

// load reads the sites in the caller's scope (narrowed to the region) and
// their approved reports for the period.
func (s *DashboardService) load(ctx context.Context, actor domain.Actor, p Period) (scope, error) {
	if !actor.Can(domain.PermSitesRead) {
		return scope{}, apperrors.ErrForbidden
	}
	all, _, err := s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Limit: overviewSiteLimit,
	})
	if err != nil {
		return scope{}, err
	}
	sc := scope{now: s.clock.Now()}
	seen := map[string]bool{}
	for _, site := range all {
		if r := strings.TrimSpace(site.Region); r != "" && !seen[strings.ToLower(r)] {
			seen[strings.ToLower(r)] = true
			sc.regions = append(sc.regions, r)
		}
		if p.Region == "" || strings.EqualFold(site.Region, p.Region) {
			sc.sites = append(sc.sites, site)
		}
	}
	if err := s.validate(p, len(sc.sites)); err != nil {
		return scope{}, err
	}
	ids := make([]string, len(sc.sites))
	for i, site := range sc.sites {
		ids[i] = site.ID
	}
	reps, err := s.reports.ApprovedInRange(ctx, ids, p.From, p.To)
	if err != nil {
		return scope{}, err
	}
	sc.byDay = make(map[string]domain.DailyReport, len(reps))
	for _, r := range reps {
		sc.byDay[ports.DayKey(r.SiteID, r.Date)] = r
	}
	if sc.adj, err = s.adjust.For(ctx, ids, p.From, p.To, sc.byDay); err != nil {
		return scope{}, err
	}
	return sc, nil
}

// daySummary is the site's figures for one day; a day with no approved report
// is the whole window unknown.
func (s *DashboardService) daySummary(sc scope, site domain.Site, date time.Time) (sum domain.DailySummary, reported bool) {
	if r, ok := sc.byDay[ports.DayKey(site.ID, date)]; ok {
		return sc.adj.summarise(s.loc, site, date, r.Intervals, true), true
	}
	return sc.adj.summarise(s.loc, site, date, nil, false), false
}

// Overview summarises every site in the caller's scope for a period.
func (s *DashboardService) Overview(ctx context.Context, actor domain.Actor, p Period) (Overview, error) {
	// The permission is checked before anything is looked up, so a kept answer
	// is never handed to someone who may not see it.
	if !actor.Can(domain.PermSitesRead) {
		return Overview{}, apperrors.ErrForbidden
	}
	return platform.Do(ctx, s.memo, "overview|"+scopeKey(actor)+"|"+fmt.Sprint(actor.Can(domain.PermFuelRead))+"|"+p.key(), func(ctx context.Context) (Overview, error) {
		return s.overview(ctx, actor, p)
	})
}

func (s *DashboardService) overview(ctx context.Context, actor domain.Actor, p Period) (Overview, error) {
	sc, err := s.load(ctx, actor, p)
	if err != nil {
		return Overview{}, err
	}
	ids := make([]string, len(sc.sites))
	for i, site := range sc.sites {
		ids[i] = site.ID
	}
	latest, err := s.reports.LatestApproved(ctx, ids)
	if err != nil {
		return Overview{}, err
	}

	yesterday := p.To.AddDate(0, 0, -1)
	statusByDay, err := s.reports.StatusForSitesOnDates(ctx, ids, []time.Time{p.To, yesterday})
	if err != nil {
		return Overview{}, err
	}
	outagesThisWeek, err := s.reports.OutageDayCount(ctx, ids, p.To.AddDate(0, 0, -6), p.To)
	if err != nil {
		return Overview{}, err
	}

	out := Overview{
		Period: p, Timezone: s.loc.String(), Regions: sc.regions, GeneratedAt: sc.now, OutagesThisWeek: outagesThisWeek,
	}
	var kpi []domain.DailySummary

	// Fuel, only for those who may see it, for the sites that have a rate.
	var fuels map[string]siteFuel
	if actor.Can(domain.PermFuelRead) {
		if fuels, err = fuelEstimates(ctx, s.fuel, s.reports, sc.sites, sc.now, s.loc); err != nil {
			return Overview{}, err
		}
	}
	// The approved reports of each site in the period, for the quality indicator.
	reportsBySite := map[string][]domain.DailyReport{}
	for _, r := range sc.byDay {
		reportsBySite[r.SiteID] = append(reportsBySite[r.SiteID], r)
	}
	var activeReports []domain.DailyReport
	activeDays := 0

	for _, site := range sc.sites {
		row := SiteRow{Site: site}
		if r, ok := statusByDay[ports.DayKey(site.ID, p.To)]; ok {
			r := r
			row.TodayReport = &r
		}
		if r, ok := statusByDay[ports.DayKey(site.ID, yesterday)]; ok {
			r := r
			row.YesterdayReport = &r
		}
		var latestPtr *domain.DailyReport
		if r, ok := latest[site.ID]; ok {
			latestPtr = &r
			if asOf, ok := r.LastObservedEnd(); ok {
				row.AsOf = &asOf
			}
		}
		row.Condition = domain.Condition(site, latestPtr, sc.now)
		if row.Condition == domain.ConditionAvailable {
			row.CurrentSource = domain.CurrentSource(latestPtr)
		}

		days := make([]domain.DailySummary, 0, p.Days())
		for date := p.From; !date.After(p.To); date = date.AddDate(0, 0, 1) {
			sum, reported := s.daySummary(sc, site, date)
			days = append(days, sum)
			if reported {
				row.DaysReported++
			}
		}
		pooled := domain.PortfolioSummary(days)
		row.DaysWithoutReport = p.Days() - row.DaysReported
		row.Quality = domain.AssessQuality(p.Days(), reportsBySite[site.ID], pooled)
		row.SLA = s.sla.Assess(pooled)
		if f, ok := fuels[site.ID]; ok && f.Estimate.Available {
			row.Fuel = &FuelBrief{
				Litres: f.Estimate.Litres, HoursRemaining: f.Estimate.HoursRemaining, Low: f.Estimate.Low(s.fuelLowHours),
				Empty: f.Estimate.EstimatedEmpty, Provisional: f.Estimate.Provisional,
			}
		}
		// A site that is available by default has a figure even if nobody
		// reported; the days without a report are still counted and shown beside it.
		if row.DaysReported > 0 || site.DefaultAvailable() {
			row.Summary = &pooled
		}

		switch site.Status {
		case domain.SiteActive:
			out.Counts.Active++
			// Only active sites enter the service KPIs; pending sites stay
			// visible separately.
			kpi = append(kpi, pooled)
			out.SiteDaysWithoutReport += row.DaysWithoutReport
			activeReports = append(activeReports, reportsBySite[site.ID]...)
			activeDays += p.Days()
			if row.DaysReported > 0 {
				out.SitesReported++
			} else {
				out.SitesNotReported++
			}
			switch row.Condition {
			case domain.ConditionAwaitingUpdate:
				out.SitesAwaitingUpdate++
			case domain.ConditionOutageReported:
				out.SitesOutageReported++
			}
		case domain.SitePendingAcceptance:
			out.Counts.PendingAcceptance++
		case domain.SiteSuspended:
			out.Counts.Suspended++
		}
		out.Sites = append(out.Sites, row)
	}
	out.Portfolio = domain.PortfolioSummary(kpi)
	out.Quality = domain.AssessQuality(activeDays, activeReports, out.Portfolio)
	out.SLA = s.sla.Assess(out.Portfolio)
	return out, nil
}

// Trend is the active portfolio's pooled figures for each day of the period.
// A day on which nothing was reported has no availability, so a chart shows a
// gap there rather than a line at zero or at 100.
func (s *DashboardService) Trend(ctx context.Context, actor domain.Actor, p Period, grain Grain) (Trend, error) {
	if !actor.Can(domain.PermSitesRead) {
		return Trend{}, apperrors.ErrForbidden
	}
	return platform.Do(ctx, s.memo, "trend|"+scopeKey(actor)+"|"+string(grain)+"|"+p.key(), func(ctx context.Context) (Trend, error) {
		return s.trend(ctx, actor, p, grain)
	})
}

func (s *DashboardService) trend(ctx context.Context, actor domain.Actor, p Period, grain Grain) (Trend, error) {
	sc, err := s.load(ctx, actor, p)
	if err != nil {
		return Trend{}, err
	}
	out := Trend{
		Period: p, Timezone: s.loc.String(), Formula: formulaDailyAvailability, Version: formulaVersion,
		GeneratedAt: sc.now, Grain: grain,
	}
	active := 0
	for _, site := range sc.sites {
		if site.Status == domain.SiteActive {
			active++
		}
	}

	type bucket struct {
		pt       TrendPoint
		sums     []domain.DailySummary
		reported map[string]bool
	}
	var order []*bucket
	byStart := map[time.Time]*bucket{}
	for date := p.From; !date.After(p.To); date = date.AddDate(0, 0, 1) {
		start := bucketStart(date, grain)
		b := byStart[start]
		if b == nil {
			b = &bucket{pt: TrendPoint{Date: start, From: date, ActiveSites: active}, reported: map[string]bool{}}
			byStart[start] = b
			order = append(order, b)
		}
		b.pt.To = date
		b.pt.Days++
		for _, site := range sc.sites {
			if site.Status != domain.SiteActive {
				continue
			}
			sum, reported := s.daySummary(sc, site, date)
			b.sums = append(b.sums, sum)
			b.pt.SiteDays++
			if reported {
				b.pt.SiteDaysReported++
				b.reported[site.ID] = true
			}
		}
	}
	out.Points = make([]TrendPoint, 0, len(order))
	for _, b := range order {
		b.pt.SitesReported = len(b.reported)
		b.pt.Summary = domain.PortfolioSummary(b.sums)
		out.Points = append(out.Points, b.pt)
	}
	return out, nil
}

const (
	mySitesWindowDays    = 30
	mySitesBarDays       = 7
	mySitesIncidentLimit = 1000
)

// DayBarEntry is one day of a site's trailing-days strip.
type DayBarEntry struct {
	Date  time.Time
	Color domain.DayBarColor
}

// MySiteRow is one card on the reporter's "my sites" grid.
type MySiteRow struct {
	Site domain.Site
	// TodayStatus is the report that best represents today's live status
	// (in-progress if there is one, else approved); nil when nothing has
	// ever been filed today.
	TodayStatus *domain.DailyReport
	// AvailabilityPct is the pooled reported availability over the trailing
	// mySitesWindowDays. Nil when the site has no approved report in the
	// window -- never 0.
	AvailabilityPct *float64
	// Bars is the trailing mySitesBarDays, oldest first.
	Bars          []DayBarEntry
	OpenIncidents int
}

type MySites struct {
	Date        time.Time
	Timezone    string
	GeneratedAt time.Time
	Sites       []MySiteRow
}

// MySites builds the reporter's own site grid: today's status, a trailing
// 7-day health strip, pooled 30-day availability and open-incident count per
// site, in one pass -- no N+1 across the actor's assigned sites.
func (s *DashboardService) MySites(ctx context.Context, actor domain.Actor) (MySites, error) {
	if !actor.Can(domain.PermSitesRead) {
		return MySites{}, apperrors.ErrForbidden
	}
	sites, _, err := s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Limit: overviewSiteLimit,
	})
	if err != nil {
		return MySites{}, err
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}

	now := s.clock.Now()
	y, m, d := now.In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -(mySitesWindowDays - 1))

	approved, err := s.reports.ApprovedInRange(ctx, ids, first, today)
	if err != nil {
		return MySites{}, err
	}
	bySite := make(map[string]map[string]domain.DailyReport, len(sites))
	for _, r := range approved {
		if bySite[r.SiteID] == nil {
			bySite[r.SiteID] = make(map[string]domain.DailyReport)
		}
		bySite[r.SiteID][r.Date.Format("2006-01-02")] = r
	}

	flat := make(map[string]domain.DailyReport, len(approved))
	for _, r := range approved {
		flat[ports.DayKey(r.SiteID, r.Date)] = r
	}
	adj, err := s.adjust.For(ctx, ids, first, today, flat)
	if err != nil {
		return MySites{}, err
	}

	statusByDay, err := s.reports.StatusForSitesOnDates(ctx, ids, []time.Time{today})
	if err != nil {
		return MySites{}, err
	}

	openIncidents, err := s.openIncidentCounts(ctx, ids)
	if err != nil {
		return MySites{}, err
	}

	rows := make([]MySiteRow, len(sites))
	for i, site := range sites {
		byDate := bySite[site.ID]
		pooled := make([]domain.DailySummary, 0, mySitesWindowDays)
		bars := make([]DayBarEntry, mySitesBarDays)
		for day := 0; day < mySitesWindowDays; day++ {
			date := today.AddDate(0, 0, -day)
			r, reported := byDate[date.Format("2006-01-02")]
			var intervals []domain.PowerInterval
			if reported {
				intervals = r.Intervals
			}
			sum := adj.summarise(s.loc, site, date, intervals, reported)
			pooled = append(pooled, sum)
			if day < mySitesBarDays {
				bars[mySitesBarDays-1-day] = DayBarEntry{Date: date, Color: domain.DayBar(reported, sum.OutageMinutes)}
			}
		}
		portfolio := domain.PortfolioSummary(pooled)
		row := MySiteRow{
			Site: site, AvailabilityPct: portfolio.ReportedAvailability, Bars: bars,
			OpenIncidents: openIncidents[site.ID],
		}
		if r, ok := statusByDay[ports.DayKey(site.ID, today)]; ok {
			r := r
			row.TodayStatus = &r
		}
		rows[i] = row
	}
	return MySites{Date: today, Timezone: s.loc.String(), GeneratedAt: now, Sites: rows}, nil
}

// openIncidentCounts fetches every open incident across the given sites in
// one query and groups it in Go, rather than one query per site.
func (s *DashboardService) openIncidentCounts(ctx context.Context, siteIDs []string) (map[string]int, error) {
	incidents, _, err := s.incidents.List(ctx, ports.IncidentFilter{
		AllSites: false, SiteIDs: siteIDs, OpenOnly: true, Limit: mySitesIncidentLimit,
	})
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(siteIDs))
	for _, inc := range incidents {
		counts[inc.SiteID]++
	}
	return counts, nil
}
