package services

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// DashboardService builds the portfolio view from approved reports only.
type DashboardService struct {
	sites     ports.SiteRepository
	reports   ports.ReportRepository
	incidents ports.IncidentRepository
	clock     ports.Clock
	loc       *time.Location
}

func NewDashboardService(
	sites ports.SiteRepository, reports ports.ReportRepository, incidents ports.IncidentRepository,
	clock ports.Clock, loc *time.Location,
) *DashboardService {
	return &DashboardService{sites: sites, reports: reports, incidents: incidents, clock: clock, loc: loc}
}

// SiteRow is one site's line on the dashboard.
type SiteRow struct {
	Site      domain.Site
	Condition domain.SiteCondition
	// AsOf is the end of the latest observed interval; nil when none.
	AsOf *time.Time
	// Summary covers the requested date, from the approved report only. Nil
	// when no approved report exists for that day, which is never shown as
	// zero or as available. This is the admin portfolio's trust-first view.
	Summary *domain.DailySummary
	// TodayReport/YesterdayReport are the report that best represents that
	// day's live status (in-progress if there is one, else approved); nil
	// when nothing has ever been filed. This is the reporter's own "did I
	// report today" view and must not be confused with Summary above: a
	// submitted-but-not-yet-approved report shows here but not in Summary.
	TodayReport, YesterdayReport *domain.DailyReport
}

type StatusCounts struct {
	Active, PendingAcceptance, Suspended int
}

type Overview struct {
	Date     time.Time
	Timezone string
	Counts   StatusCounts
	// Portfolio pools active sites by eligible minutes. A site with no
	// approved report counts its whole window as unknown.
	Portfolio           domain.DailySummary
	SitesReported       int
	SitesNotReported    int
	SitesAwaitingUpdate int
	SitesOutageReported int
	// OutagesThisWeek counts distinct site/day pairs in the trailing 7 days
	// (inclusive of Date) whose current report has an outage. An
	// approximation, not a full weekly aggregation.
	OutagesThisWeek int
	Sites           []SiteRow
	GeneratedAt     time.Time
}

const overviewSiteLimit = 1000

// Overview summarises every site in the caller's scope for one date.
func (s *DashboardService) Overview(ctx context.Context, actor domain.Actor, date time.Time) (Overview, error) {
	if !actor.Can(domain.PermSitesRead) {
		return Overview{}, apperrors.ErrForbidden
	}
	sites, _, err := s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), Limit: overviewSiteLimit,
	})
	if err != nil {
		return Overview{}, err
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}

	onDate, err := s.reports.ApprovedOnDate(ctx, ids, date)
	if err != nil {
		return Overview{}, err
	}
	latest, err := s.reports.LatestApproved(ctx, ids)
	if err != nil {
		return Overview{}, err
	}
	byDate := make(map[string]domain.DailyReport, len(onDate))
	for _, r := range onDate {
		byDate[r.SiteID] = r
	}

	yesterday := date.AddDate(0, 0, -1)
	statusByDay, err := s.reports.StatusForSitesOnDates(ctx, ids, []time.Time{date, yesterday})
	if err != nil {
		return Overview{}, err
	}
	outagesThisWeek, err := s.reports.OutageDayCount(ctx, ids, date.AddDate(0, 0, -6), date)
	if err != nil {
		return Overview{}, err
	}

	now := s.clock.Now()
	out := Overview{Date: date, Timezone: s.loc.String(), GeneratedAt: now, OutagesThisWeek: outagesThisWeek}
	var kpi []domain.DailySummary

	for _, site := range sites {
		row := SiteRow{Site: site, Condition: domain.ConditionAwaitingUpdate}
		if r, ok := statusByDay[ports.DayKey(site.ID, date)]; ok {
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
		row.Condition = domain.Condition(site, latestPtr, now)

		reported := false
		if r, ok := byDate[site.ID]; ok {
			sum := domain.Summarise(site, date, r.Intervals, s.loc)
			row.Summary = &sum
			reported = true
		}

		switch site.Status {
		case domain.SiteActive:
			out.Counts.Active++
			// Only active sites enter the service KPIs; pending sites stay
			// visible separately.
			sum := domain.Summarise(site, date, nil, s.loc)
			if row.Summary != nil {
				sum = *row.Summary
			}
			kpi = append(kpi, sum)
			if reported {
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
			sum := domain.Summarise(site, date, intervals, s.loc)
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
