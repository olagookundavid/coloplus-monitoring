package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"coloplus-backend/config"
	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/adapters/security"
	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/ports"
	"coloplus-backend/internal/services"
)

// seedPassword is shared by every sample account. It meets the real policy
// (12+ chars, a letter and a digit) so it works against the real login.
const seedPassword = "Coloplus123!"

// seed inserts a sample dataset for local development: one user per role, a
// handful of sites, daily reports moving through their workflow, an
// incident, a dispute, and sites at different acceptance stages. It goes
// through the same services the HTTP API uses, so every write still obeys
// domain validation and carries a real audit row. Safe to run more than
// once: every step looks for what already exists before creating anything.
func seed(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		URL: cfg.DB.URL, MaxConns: 2, MinConns: 0, Timezone: cfg.DB.Timezone,
	})
	if err != nil {
		return err
	}
	defer pool.Close()
	db := postgres.NewDB(pool)

	loc, err := time.LoadLocation(cfg.App.ReportingTimezone)
	if err != nil {
		return fmt.Errorf("loading reporting time zone: %w", err)
	}
	clock := platform.SystemClock{}
	audit := services.NewAuditService(postgres.NewAuditRepository(db), discardQueue{}, clock, log)
	txManager := postgres.NewTxManager(db)
	hasher := security.NewBcryptHasher(cfg.Auth.BcryptCost)

	userRepo := postgres.NewUserRepository(db)
	siteRepo := postgres.NewSiteRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	incidentRepo := postgres.NewIncidentRepository(db)
	disputeRepo := postgres.NewDisputeRepository(db)
	acceptanceRepo := postgres.NewAcceptanceRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	recoveryRepo := postgres.NewRecoveryCodeRepository(db)

	mailer, err := newMailer(cfg, log)
	if err != nil {
		return err
	}
	notifications := services.NewNotificationService(notificationRepo, mailer, audit, clock, log)

	sd := &seeder{
		log: log, clock: clock, loc: loc,
		siteRepo: siteRepo, userRepo: userRepo, reportRepo: reportRepo,
		incidentRepo: incidentRepo, disputeRepo: disputeRepo, acceptanceRepo: acceptanceRepo,
		users:      services.NewUserService(userRepo, siteRepo, hasher, txManager, audit, clock, nil, recoveryRepo, notifications),
		sites:      services.NewSiteService(siteRepo, txManager, audit),
		reports:    services.NewReportService(reportRepo, siteRepo, txManager, audit, clock, loc),
		incidents:  services.NewIncidentService(incidentRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
		disputes:   services.NewDisputeService(disputeRepo, reportRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
		acceptance: services.NewAcceptanceService(acceptanceRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
	}
	return sd.run(ctx)
}

type seeder struct {
	log   *slog.Logger
	clock ports.Clock
	loc   *time.Location

	siteRepo       ports.SiteRepository
	userRepo       ports.UserRepository
	reportRepo     ports.ReportRepository
	incidentRepo   ports.IncidentRepository
	disputeRepo    ports.DisputeRepository
	acceptanceRepo ports.AcceptanceRepository

	users      *services.UserService
	sites      *services.SiteService
	reports    *services.ReportService
	incidents  *services.IncidentService
	disputes   *services.DisputeService
	acceptance *services.AcceptanceService
}

// allPermissions is every permission the domain defines. A console seed
// tool is trusted by nature, the same way create-admin already bypasses the
// normal invite flow; this just extends that trust to the other services.
// Do not reuse this for the acceptance workflow (see acceptanceActor below).
func allPermissions() map[domain.Permission]struct{} {
	return map[domain.Permission]struct{}{
		domain.PermUsersManage:        {},
		domain.PermSitesAll:           {},
		domain.PermSitesRead:          {},
		domain.PermSitesManage:        {},
		domain.PermReportsCreate:      {},
		domain.PermReportsRead:        {},
		domain.PermReportsReview:      {},
		domain.PermPublishedRead:      {},
		domain.PermAuditRead:          {},
		domain.PermExportsCreate:      {},
		domain.PermNotificationsRead:  {},
		domain.PermIncidentsCreate:    {},
		domain.PermIncidentsRead:      {},
		domain.PermIncidentsManage:    {},
		domain.PermDisputesRead:       {},
		domain.PermDisputesRaise:      {},
		domain.PermDisputesManage:     {},
		domain.PermAcceptanceRead:     {},
		domain.PermAcceptanceManage:   {},
		domain.PermAcceptanceReview:   {},
	}
}

func seedActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: allPermissions()}
}

// AcceptanceService.Transition resolves which side is acting from the
// actor's permissions (domain.AcceptanceParty): manage -> Coloplus, review
// -> Airtel. An actor holding both would always resolve to Coloplus, so the
// acceptance workflow needs its own two actors, never the general one above.
func coloplusAcceptanceActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermAcceptanceManage: {}, domain.PermAcceptanceRead: {},
	}}
}

func airtelAcceptanceActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermAcceptanceReview: {}, domain.PermAcceptanceRead: {},
	}}
}

// reportingDate is the canonical "calendar date" value ReportService itself
// uses: today in loc, stored as a UTC midnight marker (see ReportService.today).
func reportingDate(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func addDays(date time.Time, days int) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d+days, 0, 0, 0, 0, time.UTC)
}

// localDayEnd is the exclusive end of the reporting day in loc (next day's
// local midnight) -- the same bound domain.DayBounds computes.
func localDayEnd(date time.Time, loc *time.Location) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// dailyIntervals is a simple, realistic shape: the primary source serves
// until a short morning outage, then the backup source takes over for the
// rest of the (possibly capped) day. endCap is the day's end for a past day,
// or "now" for today, since an observation can never end in the future.
func dailyIntervals(date time.Time, loc *time.Location, primary, backup domain.PowerSource, endCap time.Time) []domain.PowerInterval {
	y, m, d := date.Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, loc)
	outageStart := time.Date(y, m, d, 6, 0, 0, 0, loc)
	outageEnd := time.Date(y, m, d, 6, 30, 0, 0, loc)

	capAt := func(t time.Time) time.Time {
		if t.After(endCap) {
			return endCap
		}
		return t
	}

	var out []domain.PowerInterval
	if !endCap.After(dayStart) {
		return out
	}
	if end := capAt(outageStart); end.After(dayStart) {
		out = append(out, domain.PowerInterval{
			Start: dayStart, End: end, State: domain.StateServed, Source: primary, Basis: domain.BasisObserved,
		})
	}
	if !endCap.After(outageStart) {
		return out
	}
	if end := capAt(outageEnd); end.After(outageStart) {
		out = append(out, domain.PowerInterval{
			Start: outageStart, End: end, State: domain.StateOutage, Basis: domain.BasisObserved,
		})
	}
	if !endCap.After(outageEnd) {
		return out
	}
	out = append(out, domain.PowerInterval{
		Start: outageEnd, End: endCap, State: domain.StateServed, Source: backup, Basis: domain.BasisObserved,
	})
	return out
}

func (sd *seeder) ensureSite(ctx context.Context, actor domain.Actor, site domain.Site) (domain.Site, error) {
	existing, _, err := sd.siteRepo.List(ctx, ports.SiteFilter{AllSites: true, Query: site.AirtelCode, Limit: 10})
	if err != nil {
		return domain.Site{}, err
	}
	for _, s := range existing {
		if s.AirtelCode == site.AirtelCode {
			sd.log.Info("site already seeded", "airtel_code", s.AirtelCode, "id", s.ID)
			return s, nil
		}
	}
	created, err := sd.sites.Create(ctx, actor, site)
	if err != nil {
		return domain.Site{}, err
	}
	sd.log.Info("site created", "airtel_code", created.AirtelCode, "id", created.ID)
	return created, nil
}

func (sd *seeder) ensureUser(ctx context.Context, actor domain.Actor, email, fullName string, role domain.Role, siteIDs []string) (domain.User, error) {
	existing, err := sd.userRepo.GetByEmail(ctx, domain.NormalizeEmail(email))
	if err == nil {
		sd.log.Info("user already seeded", "email", existing.Email, "role", existing.Role)
		return existing, nil
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		return domain.User{}, err
	}
	view, err := sd.users.Create(ctx, actor, services.CreateUserInput{
		Email: email, FullName: fullName, Role: role, Password: seedPassword, SiteIDs: siteIDs,
	})
	if err != nil {
		return domain.User{}, err
	}
	sd.log.Info("user created", "email", view.User.Email, "role", view.User.Role)
	return view.User, nil
}

func (sd *seeder) ensureReportDay(
	ctx context.Context, reporter, supervisor domain.Actor, siteID string, date time.Time,
	intervals []domain.PowerInterval, targetStatus domain.ReportStatus,
) (domain.DailyReport, error) {
	existing, _, err := sd.reportRepo.List(ctx, ports.ReportFilter{AllSites: true, SiteID: siteID, From: &date, To: &date, Limit: 5})
	if err != nil {
		return domain.DailyReport{}, err
	}
	if len(existing) > 0 {
		sd.log.Info("report already seeded", "site_id", siteID, "date", date.Format("2006-01-02"))
		return existing[0], nil
	}
	created, err := sd.reports.CreateDraft(ctx, reporter, siteID, date, services.ReportInput{
		Notes: "Seeded sample entry.", Intervals: intervals,
	})
	if err != nil {
		return domain.DailyReport{}, err
	}
	if targetStatus == domain.ReportDraft {
		return created, nil
	}
	submitted, err := sd.reports.Submit(ctx, reporter, created.ID)
	if err != nil {
		return domain.DailyReport{}, err
	}
	if targetStatus == domain.ReportSubmitted {
		return submitted, nil
	}
	return sd.reports.Approve(ctx, supervisor, submitted.ID, "Looks consistent with the site's telemetry.")
}

func (sd *seeder) ensureIncident(
	ctx context.Context, reporter, supervisor domain.Actor, supervisorID, siteID, title string, startedAt time.Time,
) error {
	existing, _, err := sd.incidentRepo.List(ctx, ports.IncidentFilter{AllSites: true, SiteID: siteID, Limit: 50})
	if err != nil {
		return err
	}
	for _, inc := range existing {
		if inc.Title == title {
			sd.log.Info("incident already seeded", "site_id", siteID, "title", title)
			return nil
		}
	}
	created, err := sd.incidents.Report(ctx, reporter, services.IncidentInput{
		SiteID: siteID, Category: domain.CategoryGridLoss, Severity: domain.SeverityMajor, Impact: domain.ImpactOutage,
		Title:       title,
		Description: "Seeded sample incident: grid feed lost, generator auto-started.",
		StartedAt:   startedAt,
	})
	if err != nil {
		return err
	}
	_, err = sd.incidents.Assign(ctx, supervisor, created.ID, services.AssignInput{
		OwnerID: supervisorID, NextAction: "Confirm fuel levels and monitor restoration.",
	})
	return err
}

func (sd *seeder) ensureDispute(ctx context.Context, airtel domain.Actor, siteID, reportID, title string) error {
	existing, _, err := sd.disputeRepo.List(ctx, ports.DisputeFilter{AllSites: true, SiteID: siteID, Limit: 50})
	if err != nil {
		return err
	}
	for _, d := range existing {
		if d.Title == title {
			sd.log.Info("dispute already seeded", "site_id", siteID, "title", title)
			return nil
		}
	}
	_, err = sd.disputes.Raise(ctx, airtel, services.DisputeInput{
		SiteID: siteID, ReportID: reportID, Category: domain.DisputeWrongDuration,
		Title:       title,
		Description: "Seeded sample dispute: the recorded outage window looks shorter than observed on site.",
	})
	return err
}

// ensureAcceptanceRequested starts the process and leaves it there -- an
// "in progress" acceptance, never advanced further.
func (sd *seeder) ensureAcceptanceRequested(ctx context.Context, coloplus domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	_, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Build out generator backup and submit for inspection.",
	})
	return err
}

// ensureAcceptanceAccepted walks the full Coloplus -> Airtel sequence so one
// site ends up active (see AcceptanceService.activate).
func (sd *seeder) ensureAcceptanceAccepted(ctx context.Context, coloplus, airtel domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	if _, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Build out generator backup.",
	}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{To: domain.StageBuilt}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{
		To: domain.StageReadyForInspection, Note: "Generator and grid tie-in tested; photos and test logs are in the site folder.",
	}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, airtel, siteID, domain.AcceptanceStep{To: domain.StageInspected}); err != nil {
		return err
	}
	_, err := sd.acceptance.Transition(ctx, airtel, siteID, domain.AcceptanceStep{To: domain.StageAccepted})
	return err
}

func (sd *seeder) run(ctx context.Context) error {
	bootstrap := seedActor("system-seed")

	siteSpecs := []domain.Site{
		{
			AirtelCode: "ATL-LEK-001", ColoplusCode: "CPL-LEK-001", Name: "Lekki Toll Gate Mast",
			Address: "Lekki-Epe Expressway, Toll Gate", Region: "South West", State: "Lagos", SiteType: "Mast",
			PowerSources: []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator},
		},
		{
			AirtelCode: "ATL-IKJ-002", ColoplusCode: "CPL-IKJ-002", Name: "Ikeja Central Hub",
			Address: "Oba Akran Avenue, Ikeja", Region: "South West", State: "Lagos", SiteType: "Hub",
			PowerSources: []domain.PowerSource{domain.SourceGrid, domain.SourceSolar},
		},
		{
			AirtelCode: "ATL-GRK-003", ColoplusCode: "CPL-GRK-003", Name: "Abuja Garki Node",
			Address: "Area 3, Garki", Region: "North Central", State: "FCT", SiteType: "Node",
			PowerSources: []domain.PowerSource{domain.SourceGenerator, domain.SourceBattery},
		},
		{
			AirtelCode: "ATL-PHC-004", ColoplusCode: "CPL-PHC-004", Name: "Port Harcourt Trans-Amadi Site",
			Address: "Trans-Amadi Industrial Layout", Region: "South South", State: "Rivers", SiteType: "Tower",
			PowerSources: []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator, domain.SourceSolar},
		},
	}
	siteIDs := make([]string, len(siteSpecs))
	for i, spec := range siteSpecs {
		site, err := sd.ensureSite(ctx, bootstrap, spec)
		if err != nil {
			return fmt.Errorf("seeding site %s: %w", spec.AirtelCode, err)
		}
		siteIDs[i] = site.ID
	}
	lekkiID, ikejaID := siteIDs[0], siteIDs[1]

	if _, err := sd.ensureUser(ctx, bootstrap, "admin@coloplus.africa", "Ada Admin", domain.RoleAdmin, nil); err != nil {
		return fmt.Errorf("seeding admin: %w", err)
	}
	supervisor, err := sd.ensureUser(ctx, bootstrap, "supervisor@coloplus.africa", "Sam Supervisor", domain.RoleSupervisor, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding supervisor: %w", err)
	}
	reporter, err := sd.ensureUser(ctx, bootstrap, "reporter@coloplus.africa", "Remi Reporter", domain.RoleReporter, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding reporter: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "viewer@airtel.africa", "Vera Viewer", domain.RoleAirtelViewer, siteIDs); err != nil {
		return fmt.Errorf("seeding airtel viewer: %w", err)
	}
	reviewer, err := sd.ensureUser(ctx, bootstrap, "reviewer@airtel.africa", "Rex Reviewer", domain.RoleAirtelReviewer, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding airtel reviewer: %w", err)
	}

	reporterActor := seedActor(reporter.ID)
	supervisorActor := seedActor(supervisor.ID)
	reviewerActor := seedActor(reviewer.ID)

	// Daily reports: the two older days approved on both Lagos sites; today
	// is left in progress -- submitted on Lekki, still a draft on Ikeja --
	// so all three workflow states are visible.
	now := sd.clock.Now()
	today := reportingDate(now, sd.loc)
	yesterday := addDays(today, -1)
	dayBefore := addDays(today, -2)
	endOfToday := now.Truncate(time.Minute)

	type reportSite struct {
		id             string
		primary, backup domain.PowerSource
		todayTarget    domain.ReportStatus
	}
	reportSites := []reportSite{
		{lekkiID, domain.SourceGrid, domain.SourceGenerator, domain.ReportSubmitted},
		{ikejaID, domain.SourceGrid, domain.SourceSolar, domain.ReportDraft},
	}

	var lekkiYesterdayReportID string
	for _, rs := range reportSites {
		for _, d := range []time.Time{dayBefore, yesterday} {
			intervals := dailyIntervals(d, sd.loc, rs.primary, rs.backup, localDayEnd(d, sd.loc))
			report, err := sd.ensureReportDay(ctx, reporterActor, supervisorActor, rs.id, d, intervals, domain.ReportApproved)
			if err != nil {
				return fmt.Errorf("seeding report %s %s: %w", rs.id, d.Format("2006-01-02"), err)
			}
			if rs.id == lekkiID && d.Equal(yesterday) {
				lekkiYesterdayReportID = report.ID
			}
		}
		intervals := dailyIntervals(today, sd.loc, rs.primary, rs.backup, endOfToday)
		if _, err := sd.ensureReportDay(ctx, reporterActor, supervisorActor, rs.id, today, intervals, rs.todayTarget); err != nil {
			return fmt.Errorf("seeding today's report %s: %w", rs.id, err)
		}
	}

	// Incident: reporter reports a grid outage on Lekki; supervisor assigns
	// it to themself.
	if err := sd.ensureIncident(ctx, reporterActor, supervisorActor, supervisor.ID, lekkiID,
		"Grid outage at Lekki Toll Gate Mast", now.Add(-3*time.Hour)); err != nil {
		return fmt.Errorf("seeding incident: %w", err)
	}

	// Dispute: Airtel's reviewer disputes yesterday's approved Lekki report.
	if lekkiYesterdayReportID != "" {
		if err := sd.ensureDispute(ctx, reviewerActor, lekkiID, lekkiYesterdayReportID,
			"Reported outage duration looks short for this period"); err != nil {
			return fmt.Errorf("seeding dispute: %w", err)
		}
	}

	// Acceptance: Lekki walked all the way through to accepted (activates
	// the site); Ikeja only requested; Garki and Port Harcourt untouched,
	// still pending -- the full spread of states.
	coloplusActor := coloplusAcceptanceActor(supervisor.ID)
	airtelActor := airtelAcceptanceActor(reviewer.ID)
	if err := sd.ensureAcceptanceAccepted(ctx, coloplusActor, airtelActor, lekkiID); err != nil {
		return fmt.Errorf("seeding Lekki acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceRequested(ctx, coloplusActor, ikejaID); err != nil {
		return fmt.Errorf("seeding Ikeja acceptance: %w", err)
	}

	sd.log.Info("seed complete -- sample logins (dev only)",
		"admin", "admin@coloplus.africa",
		"supervisor", "supervisor@coloplus.africa",
		"reporter", "reporter@coloplus.africa",
		"airtel_viewer", "viewer@airtel.africa",
		"airtel_reviewer", "reviewer@airtel.africa",
		"password", seedPassword,
	)
	return nil
}
