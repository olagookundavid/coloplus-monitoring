package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// ActorSource builds the Actor a person would be on a request.
type ActorSource interface {
	ActorFor(ctx context.Context, user domain.User) (domain.Actor, error)
}

// DigestService sends each person one email a week about the last full week
// (Monday to Sunday), built from the same dashboard figures they can see in
// the app, so the digest never shows anyone more than their own screen does:
// the same site scope, published incidents only for the customer, and fuel
// only for those who may see it.
type DigestService struct {
	users     ports.UserRepository
	prefs     ports.DigestPreferenceRepository
	actors    ActorSource
	dashboard *DashboardService
	incidents ports.IncidentRepository
	notifier  *NotificationService
	clock     ports.Clock
	loc       *time.Location
	baseURL   string
}

func NewDigestService(
	users ports.UserRepository, prefs ports.DigestPreferenceRepository, actors ActorSource, dashboard *DashboardService,
	incidents ports.IncidentRepository, notifier *NotificationService, clock ports.Clock, loc *time.Location, baseURL string,
) *DigestService {
	return &DigestService{
		users: users, prefs: prefs, actors: actors, dashboard: dashboard, incidents: incidents,
		notifier: notifier, clock: clock, loc: loc, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Preference says whether the caller gets the weekly digest.
func (s *DigestService) Preference(ctx context.Context, actor domain.Actor) (bool, error) {
	if !actor.Can(domain.PermAlertsRead) {
		return false, apperrors.ErrForbidden
	}
	return s.prefs.Weekly(ctx, actor.UserID)
}

// SetPreference turns the caller's own weekly digest on or off.
func (s *DigestService) SetPreference(ctx context.Context, actor domain.Actor, on bool) error {
	if !actor.Can(domain.PermAlertsRead) {
		return apperrors.ErrForbidden
	}
	return s.prefs.SetWeekly(ctx, actor.UserID, on)
}

// lastFullWeek is the Monday to Sunday before today, in the portal's zone.
func lastFullWeek(now time.Time, loc *time.Location) (from, to time.Time) {
	y, m, d := now.In(loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	thisMonday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	return thisMonday.AddDate(0, 0, -7), thisMonday.AddDate(0, 0, -1)
}

// SendWeekly queues the digest for everyone who has not turned it off, once
// per person per week (running it again adds nothing), and returns how many
// emails it queued. A person with no sites in scope gets nothing.
func (s *DigestService) SendWeekly(ctx context.Context) (int, error) {
	from, to := lastFullWeek(s.clock.Now(), s.loc)
	week := from.Format("2006-01-02")

	users, err := s.users.List(ctx)
	if err != nil {
		return 0, err
	}
	off, err := s.prefs.OptedOut(ctx)
	if err != nil {
		return 0, err
	}
	queued := 0
	var failed []error
	for _, u := range users {
		if !u.Active || off[u.ID] {
			continue
		}
		// One person's digest failing must not stop everyone else's.
		ok, err := s.digestFor(ctx, u, week, from, to)
		if err != nil {
			failed = append(failed, fmt.Errorf("digest for a user: %w", err))
			continue
		}
		if ok {
			queued++
		}
	}
	return queued, errors.Join(failed...)
}

// digestFor queues one person's digest; false when they have nothing to be
// told about or it was already queued for this week.
func (s *DigestService) digestFor(ctx context.Context, u domain.User, week string, from, to time.Time) (bool, error) {
	actor, err := s.actors.ActorFor(ctx, u)
	if err != nil {
		return false, err
	}
	if !actor.Can(domain.PermSitesRead) {
		return false, nil
	}
	ov, err := s.dashboard.Overview(ctx, actor, Period{From: from, To: to})
	if err != nil {
		return false, err
	}
	if len(ov.Sites) == 0 {
		return false, nil
	}
	var open *ports.IncidentCounts
	if actor.Can(domain.PermIncidentsRead) {
		c, err := s.incidents.Counts(ctx, ports.IncidentFilter{
			AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), OpenOnly: true, PublishedOnly: actor.Can(domain.PermPublishedRead),
		}, s.clock.Now())
		if err != nil {
			return false, err
		}
		open = &c
	}
	subject, body := weeklyDigestEmail(ov, open, actor.Can(domain.PermPublishedRead), s.baseURL)
	return s.notifier.Enqueue(ctx, domain.Notification{
		Kind: "digest.weekly", Recipient: u.Email, Subject: subject, Body: body,
		DedupeKey: "weekly-digest:" + week + ":" + u.ID,
	})
}

func digestPct(v *float64) string {
	if v == nil {
		return "not available"
	}
	return fmt.Sprintf("%.2f%%", *v)
}

// weeklyDigestEmail words the week. Availability is always stated with how
// much of it rests on reports and how much is assumed, as everywhere else.
func weeklyDigestEmail(ov Overview, open *ports.IncidentCounts, customer bool, baseURL string) (subject, body string) {
	from, to := ov.Period.From.Format("2 Jan"), ov.Period.To.Format("2 Jan 2006")
	subject = fmt.Sprintf("Coloplus weekly summary, %s to %s", from, to)

	var b strings.Builder
	fmt.Fprintf(&b, "Your week, %s to %s, for %d sites.\n\n", from, to, len(ov.Sites))
	p := ov.Portfolio
	fmt.Fprintf(&b, "Availability as reported: %s\n", digestPct(p.ReportedAvailability))
	fmt.Fprintf(&b, "Confirmed by reports: %s of the eligible time.\n", digestPct(p.ConfirmedServedShare))
	if p.AssumedMinutes > 0 {
		fmt.Fprintf(&b, "Assumed available because nobody reported: %.1f hours. This is an assumption, not a confirmation.\n", float64(p.AssumedMinutes)/60)
	}
	fmt.Fprintf(&b, "Days without an approved report: %d site-days.\n", ov.SiteDaysWithoutReport)
	fmt.Fprintf(&b, "Data quality: %s.\n", ov.Quality.Level)

	type row struct {
		label string
		mins  int
	}
	var outages []row
	for _, r := range ov.Sites {
		if r.Summary != nil && r.Summary.OutageMinutes > 0 {
			outages = append(outages, row{siteLabel(r.Site), r.Summary.OutageMinutes})
		}
	}
	sort.Slice(outages, func(i, j int) bool { return outages[i].mins > outages[j].mins })
	b.WriteString("\nOutages\n")
	if len(outages) == 0 {
		b.WriteString("No outage was reported this week.\n")
	} else {
		fmt.Fprintf(&b, "%d site(s) reported an outage. The longest:\n", len(outages))
		for i, o := range outages {
			if i == 5 {
				break
			}
			fmt.Fprintf(&b, "  - %s: %.1f hours\n", o.label, float64(o.mins)/60)
		}
	}

	if open != nil {
		b.WriteString("\nIncidents\n")
		if customer {
			fmt.Fprintf(&b, "%d open incident(s) shared with you.\n", open.Open)
		} else {
			fmt.Fprintf(&b, "%d open, %d critical, %d overdue for an update.\n", open.Open, open.Critical, open.Overdue)
		}
	}

	var waiting, fuel []string
	for _, r := range ov.Sites {
		if r.Condition == domain.ConditionAwaitingUpdate {
			waiting = append(waiting, siteLabel(r.Site))
		}
		if r.Fuel != nil && (r.Fuel.Low || r.Fuel.Empty) {
			fuel = append(fuel, siteLabel(r.Site))
		}
	}
	list := func(title string, items []string, note string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n%s (%d)%s\n", title, len(items), note)
		for i, it := range items {
			if i == 10 {
				fmt.Fprintf(&b, "  ... and %d more\n", len(items)-10)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", it)
		}
	}
	list("Waiting for a report update", waiting, " (late data, not an outage)")
	list("Fuel low or probably empty", fuel, " (estimates)")

	fmt.Fprintf(&b, "\nOpen the dashboard: %s\n\nYou can turn this email off in your notification settings.\n", baseURL)
	return subject, b.String()
}
