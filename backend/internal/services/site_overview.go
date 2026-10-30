package services

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// SiteOverviewService gathers what the site screen shows in one call: the
// profile, the current condition, the last days, and the open incidents,
// disputes and acceptance. Each section is only included for people allowed
// to see it, and everything is read through the same scope rules as the
// individual endpoints.
type SiteOverviewService struct {
	sites       ports.SiteRepository
	reports     ports.ReportRepository
	incidents   ports.IncidentRepository
	disputes    ports.DisputeRepository
	acceptances ports.AcceptanceRepository
	clock       ports.Clock
	loc         *time.Location
}

func NewSiteOverviewService(
	sites ports.SiteRepository, reports ports.ReportRepository, incidents ports.IncidentRepository,
	disputes ports.DisputeRepository, acceptances ports.AcceptanceRepository, clock ports.Clock, loc *time.Location,
) *SiteOverviewService {
	return &SiteOverviewService{
		sites: sites, reports: reports, incidents: incidents, disputes: disputes, acceptances: acceptances,
		clock: clock, loc: loc,
	}
}

const (
	defaultOverviewDays = 14
	maxOverviewDays     = 90
	overviewListLimit   = 20
)

// SiteDay is one reporting day. A day with no approved report is a visible
// gap: it has no summary, and is never shown as available or as zero.
type SiteDay struct {
	Date         time.Time
	ReportID     string
	Revision     int
	Summary      *domain.DailySummary
	OpenDisputes int
}

type SiteOverview struct {
	Site      domain.Site
	Condition domain.SiteCondition
	// AsOf is the end of the latest observed interval; nil when none.
	AsOf *time.Time
	Days []SiteDay

	// The sections below are nil when the caller may not see them.
	Incidents      []domain.Incident
	IncidentsTotal int
	ShowIncidents  bool
	Disputes       []domain.Dispute
	DisputesTotal  int
	ShowDisputes   bool
	Acceptance     *domain.Acceptance
}

// Overview builds the site screen for the last `days` reporting days.
func (s *SiteOverviewService) Overview(ctx context.Context, actor domain.Actor, siteID string, days int) (SiteOverview, error) {
	if !actor.Can(domain.PermSitesRead) {
		return SiteOverview{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return SiteOverview{}, apperrors.ErrNotFound
	}
	if days <= 0 {
		days = defaultOverviewDays
	}
	if days > maxOverviewDays {
		days = maxOverviewDays
	}
	site, err := s.sites.GetByID(ctx, siteID)
	if err != nil {
		return SiteOverview{}, err
	}

	now := s.clock.Now()
	out := SiteOverview{Site: site}

	latest, err := s.reports.LatestApproved(ctx, []string{siteID})
	if err != nil {
		return SiteOverview{}, err
	}
	var latestPtr *domain.DailyReport
	if r, ok := latest[siteID]; ok {
		latestPtr = &r
		if asOf, ok := r.LastObservedEnd(); ok {
			out.AsOf = &asOf
		}
	}
	out.Condition = domain.Condition(site, latestPtr, now)

	y, m, d := now.In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -(days - 1))
	reps, err := s.reports.ApprovedBetween(ctx, siteID, first, today)
	if err != nil {
		return SiteOverview{}, err
	}
	byDate := make(map[string]domain.DailyReport, len(reps))
	ids := make([]string, len(reps))
	for i, r := range reps {
		byDate[r.Date.Format("2006-01-02")] = r
		ids[i] = r.ID
	}
	var disputed map[string]int
	if actor.Can(domain.PermDisputesRead) {
		if disputed, err = s.disputes.OpenByReport(ctx, ids); err != nil {
			return SiteOverview{}, err
		}
	}
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i)
		day := SiteDay{Date: date}
		if r, ok := byDate[date.Format("2006-01-02")]; ok {
			sum := domain.Summarise(site, date, r.Intervals, s.loc)
			day.ReportID, day.Revision, day.Summary, day.OpenDisputes = r.ID, r.Revision, &sum, disputed[r.ID]
		}
		out.Days = append(out.Days, day)
	}

	if actor.Can(domain.PermIncidentsRead) || actor.Can(domain.PermPublishedRead) {
		out.ShowIncidents = true
		out.Incidents, out.IncidentsTotal, err = s.incidents.List(ctx, ports.IncidentFilter{
			SiteID: siteID, AllSites: true, OpenOnly: true, PublishedOnly: !actor.Can(domain.PermIncidentsRead),
			Limit: overviewListLimit,
		})
		if err != nil {
			return SiteOverview{}, err
		}
	}
	if actor.Can(domain.PermDisputesRead) {
		out.ShowDisputes = true
		out.Disputes, out.DisputesTotal, err = s.disputes.List(ctx, ports.DisputeFilter{
			SiteID: siteID, AllSites: true, OpenOnly: true, Limit: overviewListLimit,
		})
		if err != nil {
			return SiteOverview{}, err
		}
	}
	if actor.Can(domain.PermAcceptanceRead) {
		a, err := s.acceptances.GetBySite(ctx, siteID, false)
		switch {
		case err == nil:
			out.Acceptance = &a
		case !isNotFound(err):
			return SiteOverview{}, err
		}
	}
	return out, nil
}
