package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"
)

// ReportAlertService tells the people who owe a daily report that one is
// overdue. It uses the same rule as the dashboard (domain.Condition), so an
// email is sent exactly when a site shows "awaiting update". A late report is
// stale data, not an outage, and the email says so.
type ReportAlertService struct {
	sites    ports.SiteRepository
	reports  ports.ReportRepository
	users    ports.UserRepository
	notifier *NotificationService
	clock    ports.Clock
	loc      *time.Location
	baseURL  string
}

func NewReportAlertService(
	sites ports.SiteRepository, reports ports.ReportRepository, users ports.UserRepository,
	notifier *NotificationService, clock ports.Clock, loc *time.Location, baseURL string,
) *ReportAlertService {
	return &ReportAlertService{
		sites: sites, reports: reports, users: users, notifier: notifier, clock: clock, loc: loc,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// AlertOverdue emails the site's reporters and supervisors once per site per
// day while the site stays awaiting an update, and returns how many emails it
// queued. A site that has not yet been around for its freshness window is
// left alone, so a new site is not nagged before its first report is due.
func (s *ReportAlertService) AlertOverdue(ctx context.Context) (int, error) {
	now := s.clock.Now()
	sites, _, err := s.sites.List(ctx, ports.SiteFilter{AllSites: true, Status: domain.SiteActive, Limit: overviewSiteLimit})
	if err != nil {
		return 0, err
	}
	ids := make([]string, len(sites))
	for i, site := range sites {
		ids[i] = site.ID
	}
	latest, err := s.reports.LatestApproved(ctx, ids)
	if err != nil {
		return 0, err
	}

	day := now.In(s.loc).Format("2006-01-02")
	queued := 0
	for _, site := range sites {
		var latestPtr *domain.DailyReport
		if r, ok := latest[site.ID]; ok {
			latestPtr = &r
		}
		if domain.Condition(site, latestPtr, now) != domain.ConditionAwaitingUpdate {
			continue
		}
		if latestPtr == nil && now.Sub(site.CreatedAt) < time.Duration(site.FreshnessHours)*time.Hour {
			continue
		}
		recipients, err := s.recipients(ctx, site.ID)
		if err != nil {
			return queued, err
		}
		var asOf *time.Time
		if latestPtr != nil {
			if t, ok := latestPtr.LastObservedEnd(); ok {
				asOf = &t
			}
		}
		subject, body := reportOverdueEmail(site, asOf, s.loc, s.baseURL+"/sites/"+site.ID)
		n, err := s.notifier.EnqueueFor(ctx, recipients, "report.overdue", fmt.Sprintf("report-overdue:%s:%s", site.ID, day), subject, body)
		queued += n
		if err != nil {
			return queued, err
		}
	}
	return queued, nil
}

// recipients are the people who write and approve the site's reports.
func (s *ReportAlertService) recipients(ctx context.Context, siteID string) ([]domain.User, error) {
	var out []domain.User
	for _, perm := range []domain.Permission{domain.PermReportsCreate, domain.PermReportsReview} {
		users, err := s.users.UsersWithPermission(ctx, perm, siteID)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			out = appendUnique(out, u)
		}
	}
	return out, nil
}
