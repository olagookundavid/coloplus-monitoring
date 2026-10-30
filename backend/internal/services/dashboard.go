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
	// Summary covers the requested date. Nil when no approved report exists
	// for that day, which is never shown as zero or as available.
	Summary *domain.DailySummary
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
	Sites               []SiteRow
	GeneratedAt         time.Time
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

	now := s.clock.Now()
	out := Overview{Date: date, Timezone: s.loc.String(), GeneratedAt: now}
	var kpi []domain.DailySummary

	for _, site := range sites {
		row := SiteRow{Site: site, Condition: domain.ConditionAwaitingUpdate}
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
