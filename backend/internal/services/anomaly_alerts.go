package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"
)

// anomalyWindowDays is how many finished days back the check looks. A report
// is only ever flagged once (the alert is keyed by the report), so looking back
// a few days catches a report approved late without repeating anything.
const anomalyWindowDays = 3

// AnomalyService flags approved reports whose numbers do not fit what is known
// about the site, so someone looks at them. It asserts nothing about why: a
// flag is "worth a look", never an accusation, and it changes no figure.
//
// The rules are deliberately few and plain (DECISIONS 066):
//   - generator hours at a site that has no generator;
//   - more generator hours than the site's service window has;
//   - an outage with no incident opened around that day.
type AnomalyService struct {
	sites     ports.SiteRepository
	reports   ports.ReportRepository
	incidents ports.IncidentRepository
	users     ports.UserRepository
	notifier  *NotificationService
	clock     ports.Clock
	loc       *time.Location
	baseURL   string
}

func NewAnomalyService(
	sites ports.SiteRepository, reports ports.ReportRepository, incidents ports.IncidentRepository, users ports.UserRepository,
	notifier *NotificationService, clock ports.Clock, loc *time.Location, baseURL string,
) *AnomalyService {
	return &AnomalyService{
		sites: sites, reports: reports, incidents: incidents, users: users, notifier: notifier,
		clock: clock, loc: loc, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

type anomaly struct {
	code, text string
}

// checkReport applies the rules to one approved report. hasIncident says an
// incident was opened at the site around the report's day.
func checkReport(site domain.Site, r domain.DailyReport, hasIncident bool, loc *time.Location) []anomaly {
	var out []anomaly
	hours := domain.GeneratorHoursOf(r)
	window := float64(site.ServiceEndMinute-site.ServiceStartMinute) / 60
	if hours > 0 && !site.HasSource(domain.SourceGenerator) {
		out = append(out, anomaly{"generator-not-installed",
			fmt.Sprintf("%.1f generator hours were reported, but the site has no generator installed.", hours)})
	}
	if hours > window+0.01 {
		out = append(out, anomaly{"generator-hours-exceed-window",
			fmt.Sprintf("%.1f generator hours were reported, more than the %.1f hours the site is serviced in a day.", hours, window)})
	}
	if domain.Summarise(site, r.Date, r.Intervals, loc).OutageMinutes > 0 && !hasIncident {
		out = append(out, anomaly{"outage-without-incident",
			"An outage was reported, but no incident was opened for the site around that day."})
	}
	return out
}

// Check looks at the last few finished days and queues one email (and one alert)
// per flagged report to the people who review the site's reports. It returns how
// many emails it queued.
func (s *AnomalyService) Check(ctx context.Context) (int, error) {
	now := s.clock.Now()
	y, m, d := now.In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	from, to := today.AddDate(0, 0, -anomalyWindowDays), today.AddDate(0, 0, -1)

	sites, err := listAllActiveSites(ctx, s.sites)
	if err != nil {
		return 0, err
	}
	ids := make([]string, len(sites))
	bySite := make(map[string]domain.Site, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
		bySite[site.ID] = site
	}
	reports, err := s.reports.ApprovedInRange(ctx, ids, from, to)
	if err != nil {
		return 0, err
	}

	// Incidents that started from a day before the window to a day after it,
	// grouped by site, so "around that day" is one query rather than one per report.
	start, _ := domain.DayBounds(from.AddDate(0, 0, -1), s.loc)
	_, end := domain.DayBounds(to.AddDate(0, 0, 1), s.loc)
	incs, _, err := s.incidents.List(ctx, ports.IncidentFilter{
		AllSites: true, StartedFrom: &start, StartedBefore: &end, Limit: maxExportRows,
	})
	if err != nil {
		return 0, err
	}
	started := map[string][]time.Time{}
	for _, i := range incs {
		started[i.SiteID] = append(started[i.SiteID], i.StartedAt)
	}

	queued := 0
	for _, r := range reports {
		site, ok := bySite[r.SiteID]
		if !ok {
			continue
		}
		dayStart, _ := domain.DayBounds(r.Date, s.loc)
		lo, hi := dayStart.AddDate(0, 0, -1), dayStart.AddDate(0, 0, 2)
		has := false
		for _, t := range started[site.ID] {
			if !t.Before(lo) && t.Before(hi) {
				has = true
				break
			}
		}
		for _, a := range checkReport(site, r, has, s.loc) {
			recipients, err := s.users.UsersWithPermission(ctx, domain.PermReportsReview, site.ID)
			if err != nil {
				return queued, err
			}
			subject := fmt.Sprintf("Worth a look: %s, %s", siteLabel(site), r.Date.Format("2 Jan"))
			body := fmt.Sprintf("%s\n\nThis is a check on the numbers, not a finding: there may be a good reason. Nothing has been changed.\n\nSite: %s\nDay: %s\n\nOpen the site: %s\n",
				a.text, siteLabel(site), r.Date.Format("2 January 2006"), s.baseURL+"/sites/"+site.ID)
			n, err := s.notifier.EnqueueFor(ctx, recipients, "report.anomaly", "anomaly:"+a.code+":"+r.ID, subject, body)
			if err != nil {
				return queued, err
			}
			queued += n
		}
	}
	return queued, nil
}
