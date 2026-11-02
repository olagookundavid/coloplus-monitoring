package services

import (
	"context"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// DashboardService builds the portfolio view from approved reports only.
type DashboardService struct {
	sites   ports.SiteRepository
	reports ports.ReportRepository
	clock   ports.Clock
	loc     *time.Location
}

func NewDashboardService(sites ports.SiteRepository, reports ports.ReportRepository, clock ports.Clock, loc *time.Location) *DashboardService {
	return &DashboardService{sites: sites, reports: reports, clock: clock, loc: loc}
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
	// DaysReported is how many days of the period have an approved report.
	DaysReported int
}

type StatusCounts struct {
	Active, PendingAcceptance, Suspended int
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
	Sites               []SiteRow
	GeneratedAt         time.Time
}

// TrendPoint is the portfolio on one reporting day.
type TrendPoint struct {
	Date time.Time
	// ActiveSites is how many active sites were expected to report;
	// SitesReported how many have an approved report for the day.
	ActiveSites, SitesReported int
	Summary                    domain.DailySummary
}

type Trend struct {
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
	return sc, nil
}

// daySummary is the site's figures for one day; a day with no approved report
// is the whole window unknown.
func (s *DashboardService) daySummary(sc scope, site domain.Site, date time.Time) (sum domain.DailySummary, reported bool) {
	if r, ok := sc.byDay[ports.DayKey(site.ID, date)]; ok {
		return domain.Summarise(site, date, r.Intervals, s.loc), true
	}
	return domain.Summarise(site, date, nil, s.loc), false
}

// Overview summarises every site in the caller's scope for a period.
func (s *DashboardService) Overview(ctx context.Context, actor domain.Actor, p Period) (Overview, error) {
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

	out := Overview{Period: p, Timezone: s.loc.String(), Regions: sc.regions, GeneratedAt: sc.now}
	var kpi []domain.DailySummary

	for _, site := range sc.sites {
		row := SiteRow{Site: site}
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
		if row.DaysReported > 0 {
			row.Summary = &pooled
		}

		switch site.Status {
		case domain.SiteActive:
			out.Counts.Active++
			// Only active sites enter the service KPIs; pending sites stay
			// visible separately.
			kpi = append(kpi, pooled)
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
	return out, nil
}

// Trend is the active portfolio's pooled figures for each day of the period.
// A day on which nothing was reported has no availability, so a chart shows a
// gap there rather than a line at zero or at 100.
func (s *DashboardService) Trend(ctx context.Context, actor domain.Actor, p Period) (Trend, error) {
	sc, err := s.load(ctx, actor, p)
	if err != nil {
		return Trend{}, err
	}
	out := Trend{
		Period: p, Timezone: s.loc.String(), Formula: formulaDailyAvailability, Version: formulaVersion,
		GeneratedAt: sc.now, Points: make([]TrendPoint, 0, p.Days()),
	}
	for date := p.From; !date.After(p.To); date = date.AddDate(0, 0, 1) {
		pt := TrendPoint{Date: date}
		var sums []domain.DailySummary
		for _, site := range sc.sites {
			if site.Status != domain.SiteActive {
				continue
			}
			sum, reported := s.daySummary(sc, site, date)
			sums = append(sums, sum)
			pt.ActiveSites++
			if reported {
				pt.SitesReported++
			}
		}
		pt.Summary = domain.PortfolioSummary(sums)
		out.Points = append(out.Points, pt)
	}
	return out, nil
}
