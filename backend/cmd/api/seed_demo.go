package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"coloplus-backend/config"
	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// seedDemo attaches a small, believable history to people who have already
// been invited, so a test session starts with something to look at. It is the
// production-safe counterpart of `api seed`: it creates no users, sets no
// password and touches no site that is not one of the TEST- sites. It goes
// through the same services as the API, so every rule still holds and every
// action has a real audit row.
//
//	api seed-demo --reporter=<email> --supervisors=<email>,<email> --reviewer=<email>
//
// What it leaves behind is described on the plan page. Two things to know:
// the audit log names the invited people as the authors of that history
// although they did not do it, and no email is sent for any of it. Today is
// left empty on purpose, so the testers can enter today's reports themselves.
// Safe to run more than once: each step looks for what already exists.
func seedDemo(ctx context.Context, cfg config.Config, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	reporterEmail := fs.String("reporter", "", "email of the invited field reporter")
	supervisorEmails := fs.String("supervisors", "", "emails of two invited supervisors, separated by a comma")
	reviewerEmail := fs.String("reviewer", "", "email of the invited customer reviewer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var supervisors []string
	for _, e := range strings.Split(*supervisorEmails, ",") {
		if e = strings.TrimSpace(e); e != "" {
			supervisors = append(supervisors, e)
		}
	}
	const usage = "usage: api seed-demo --reporter=<email> --supervisors=<email>,<email> --reviewer=<email>"
	if strings.TrimSpace(*reporterEmail) == "" || strings.TrimSpace(*reviewerEmail) == "" || len(supervisors) != 2 {
		return errors.New(usage)
	}
	if domain.NormalizeEmail(supervisors[0]) == domain.NormalizeEmail(supervisors[1]) {
		return errors.New("the two supervisors need different email addresses: a second person has to approve and verify")
	}

	sd, closeDB, err := newSeeder(ctx, cfg, log, true)
	if err != nil {
		return err
	}
	defer closeDB()
	return sd.runDemo(ctx, strings.TrimSpace(*reporterEmail), supervisors[0], supervisors[1], strings.TrimSpace(*reviewerEmail))
}

// demoUser finds an already-invited person and checks the role is the one the
// demo needs them in.
func (sd *seeder) demoUser(ctx context.Context, email string, role domain.Role) (domain.User, error) {
	u, err := sd.userRepo.GetByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, apperrors.ErrNotFound) {
		return domain.User{}, fmt.Errorf("%s has not been invited yet: invite them from the Users screen first", email)
	}
	if err != nil {
		return domain.User{}, err
	}
	if u.Role != role {
		return domain.User{}, fmt.Errorf("%s has the role %s, but the demo needs a %s there", email, u.Role, role)
	}
	return u, nil
}

// demoSite finds one of the TEST- sites by its customer code.
func (sd *seeder) demoSite(ctx context.Context, code string) (domain.Site, error) {
	list, _, err := sd.siteRepo.List(ctx, ports.SiteFilter{AllSites: true, Query: code, Limit: 10})
	if err != nil {
		return domain.Site{}, err
	}
	for _, s := range list {
		if s.CustomerSiteCode == code {
			return s, nil
		}
	}
	return domain.Site{}, fmt.Errorf("site %s does not exist: import the test sites first", code)
}

// needAssigned refuses to go on when a staff member is not assigned to a demo
// site: an incident owner is checked against the real assignment, and a
// person who cannot see the site could not see what is made for them.
func (sd *seeder) needAssigned(ctx context.Context, u domain.User, sites map[string]domain.Site) error {
	have, err := sd.userRepo.SiteIDsForUser(ctx, u.ID)
	if err != nil {
		return err
	}
	assigned := make(map[string]struct{}, len(have))
	for _, id := range have {
		assigned[id] = struct{}{}
	}
	var missing []string
	for code, s := range sites {
		if _, ok := assigned[s.ID]; !ok {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s is not assigned to %s: give them the sites first (Users, site assignment)", u.Email, strings.Join(missing, ", "))
	}
	return nil
}

func (sd *seeder) runDemo(ctx context.Context, reporterEmail, supervisorEmail, supervisor2Email, reviewerEmail string) error {
	reporter, err := sd.demoUser(ctx, reporterEmail, domain.RoleReporter)
	if err != nil {
		return err
	}
	supervisor, err := sd.demoUser(ctx, supervisorEmail, domain.RoleSupervisor)
	if err != nil {
		return err
	}
	supervisor2, err := sd.demoUser(ctx, supervisor2Email, domain.RoleSupervisor)
	if err != nil {
		return err
	}
	reviewer, err := sd.demoUser(ctx, reviewerEmail, domain.RoleReviewer)
	if err != nil {
		return err
	}

	sites := map[string]domain.Site{}
	for _, code := range []string{"TEST-LAG-001", "TEST-LAG-002", "TEST-LAG-003"} {
		s, err := sd.demoSite(ctx, code)
		if err != nil {
			return err
		}
		sites[code] = s
	}
	for _, u := range []domain.User{reporter, supervisor, supervisor2} {
		if err := sd.needAssigned(ctx, u, sites); err != nil {
			return err
		}
	}
	if scope, err := sd.userRepo.SiteIDsForUser(ctx, reviewer.ID); err == nil && len(scope) == 0 {
		sd.log.Warn("the reviewer has no site scope yet, so they will see none of this until one is set", "email", reviewer.Email)
	}
	lag1, lag2, lag3 := sites["TEST-LAG-001"], sites["TEST-LAG-002"], sites["TEST-LAG-003"]

	reporterActor := seedActor(reporter.ID)
	supervisorActor := seedActor(supervisor.ID)
	supervisor2Actor := seedActor(supervisor2.ID)

	// Past days only. Today is left empty for the testers.
	now := sd.clock.Now()
	today := reportingDate(now, sd.loc)
	day := func(back int) time.Time { return addDays(today, -back) }

	type reportSpec struct {
		site            domain.Site
		back            int
		primary, backup domain.PowerSource
		target          domain.ReportStatus
	}
	reportSpecs := []reportSpec{
		{lag1, 1, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag1, 2, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag1, 3, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag2, 1, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag2, 2, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag2, 3, domain.SourceGrid, domain.SourceGenerator, domain.ReportApproved},
		{lag3, 1, domain.SourceGrid, domain.SourceBattery, domain.ReportApproved},
		{lag3, 2, domain.SourceGrid, domain.SourceBattery, domain.ReportApproved},
		{lag3, 3, domain.SourceGrid, domain.SourceBattery, domain.ReportApproved},
		// Returned for a fix, so the reporter has something waiting on Corrections.
		{lag2, 4, domain.SourceGrid, domain.SourceGenerator, domain.ReportRejected},
	}
	reportIDs := map[string]string{} // "CODE/back" -> report id
	for _, rs := range reportSpecs {
		d := day(rs.back)
		intervals := dailyIntervals(d, sd.loc, rs.primary, rs.backup, localDayEnd(d, sd.loc))
		report, err := sd.ensureReportDay(ctx, reporterActor, supervisorActor, rs.site.ID, d, intervals, rs.target)
		if err != nil {
			return fmt.Errorf("demo report %s %s: %w", rs.site.CustomerSiteCode, d.Format("2006-01-02"), err)
		}
		reportIDs[fmt.Sprintf("%s/%d", rs.site.CustomerSiteCode, rs.back)] = report.ID
	}

	// One approved day that is then corrected end to end, so the original shows
	// as superseded. A second supervisor approves the correction.
	supersedeDay := day(6)
	supersedeSource, err := sd.ensureReportDay(ctx, reporterActor, supervisorActor, lag1.ID, supersedeDay,
		dailyIntervals(supersedeDay, sd.loc, domain.SourceGrid, domain.SourceGenerator, localDayEnd(supersedeDay, sd.loc)), domain.ReportApproved)
	if err != nil {
		return fmt.Errorf("demo report to correct: %w", err)
	}
	if err := sd.ensureSupersededReport(ctx, reporterActor, supervisor2Actor, supersedeSource.ID); err != nil {
		return fmt.Errorf("demo correction: %w", err)
	}

	// Incidents: one closed and published (the customer can see it), one being
	// worked, one just reported. Titles say "sample" so they cannot be mistaken
	// for what the testers enter.
	incidentSpecs := []incidentSpec{
		{
			siteID: lag3.ID, title: "Sample: earlier grid failure at Surulere", category: domain.CategoryGridLoss,
			severity: domain.SeverityMajor, impact: domain.ImpactOutage, startedAt: now.Add(-48 * time.Hour),
			target: domain.IncidentClosed, publish: true, ownerID: supervisor.ID,
			reporter: reporterActor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: lag2.ID, title: "Sample: generator will not start at Lekki", category: domain.CategoryGeneratorFailure,
			severity: domain.SeverityMinor, impact: domain.ImpactNoOutage, startedAt: now.Add(-5 * time.Hour),
			target: domain.IncidentInProgress, ownerID: supervisor.ID,
			reporter: reporterActor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: lag1.ID, title: "Sample: low fuel reported at Ikeja", category: domain.CategoryLowFuel,
			severity: domain.SeverityMinor, impact: domain.ImpactNoOutage, startedAt: now.Add(-2 * time.Hour),
			target: domain.IncidentReported, ownerID: supervisor.ID,
			reporter: reporterActor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
	}
	for _, spec := range incidentSpecs {
		if err := sd.ensureIncidentAt(ctx, spec); err != nil {
			return fmt.Errorf("demo incident %q: %w", spec.title, err)
		}
	}

	// Disputes and tickets: one open and one finished of each, so a supervisor
	// has something to work and the reviewer has something finished to read.
	coloplusDispute, customerDispute := coloplusDisputeActor(supervisor.ID), customerDisputeActor(reviewer.ID)
	disputeSpecs := []struct {
		siteID, reportID, title string
		target                  domain.DisputeStatus
	}{
		{lag1.ID, reportIDs["TEST-LAG-001/2"], "Sample: reported outage looks short for this day", domain.DisputeOpen},
		{lag2.ID, reportIDs["TEST-LAG-002/3"], "Sample: outage already explained", domain.DisputeClosed},
	}
	for _, spec := range disputeSpecs {
		if err := sd.ensureDisputeAt(ctx, coloplusDispute, customerDispute, spec.siteID, spec.reportID, spec.title, spec.target); err != nil {
			return fmt.Errorf("demo dispute %q: %w", spec.title, err)
		}
	}
	coloplusTicket, customerTicket := coloplusTicketActor(supervisor2.ID), customerTicketActor(reviewer.ID)
	ticketSpecs := []struct {
		siteID, subject string
		category        domain.TicketCategory
		priority        domain.TicketPriority
		target          domain.TicketStatus
	}{
		{lag1.ID, "Sample: generator making an unusual noise", domain.TicketGeneratorFault, domain.TicketMedium, domain.TicketOpen},
		{lag3.ID, "Sample: tower light not working", domain.TicketEquipmentFault, domain.TicketLow, domain.TicketResolved},
	}
	for _, spec := range ticketSpecs {
		if err := sd.ensureTicketAt(ctx, coloplusTicket, customerTicket, spec.siteID, spec.subject, spec.category, spec.priority, spec.target); err != nil {
			return fmt.Errorf("demo ticket %q: %w", spec.subject, err)
		}
	}

	// Planned maintenance at Ikeja, on days other than the ones the plan asks
	// the testers to use: one waiting for approval, one approved. The reporter
	// asks and a supervisor decides.
	maintenanceSpecs := []struct {
		reason string
		start  time.Time
		target domain.MaintenanceStatus
	}{
		{"Sample: transformer inspection", atLocalHour(addDays(today, 4), 9, sd.loc), domain.MaintenanceRequested},
		{"Sample: generator service", atLocalHour(addDays(today, 5), 9, sd.loc), domain.MaintenanceApproved},
	}
	for _, spec := range maintenanceSpecs {
		if err := sd.ensureMaintenanceWindow(ctx, reporterActor, supervisorActor, lag1.ID, spec.reason, spec.start, spec.target); err != nil {
			return fmt.Errorf("demo maintenance %q: %w", spec.reason, err)
		}
	}

	sd.log.Info("demo history is in place",
		"reporter", reporter.Email, "supervisors", supervisor.Email+", "+supervisor2.Email, "reviewer", reviewer.Email,
		"sites", "TEST-LAG-001, TEST-LAG-002, TEST-LAG-003", "today", "left empty for the testers")
	return nil
}
