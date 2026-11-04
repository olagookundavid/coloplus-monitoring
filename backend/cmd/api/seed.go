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

// seed inserts a sample dataset for local development: two users per role, six
// sites, and every other table populated with a spread of the states each one
// can be in -- reports from draft through superseded, incidents from reported
// through closed and published, disputes through their full workflow, support
// tickets, planned maintenance, fuel entries (including a voided one and a
// variance), a stored calculation run and a closed (and a closed-then-reopened)
// reporting month. It goes through the same services the HTTP API uses, so
// every write still obeys domain validation and carries a real audit row.
// Safe to run more than once: every step looks for what already exists before
// creating anything.
//
// Not seeded: evidence attachments, because R2 (file storage) is not
// configured for local development -- the upload endpoints answer 503
// without it, and a DB row with no real stored file would be misleading.
func seed(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		URL: cfg.DB.URL, MaxConns: 4, MinConns: 0, Timezone: cfg.DB.Timezone,
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
	incidentCorrectionRepo := postgres.NewIncidentCorrectionRepository(db)
	disputeRepo := postgres.NewDisputeRepository(db)
	ticketRepo := postgres.NewTicketRepository(db)
	acceptanceRepo := postgres.NewAcceptanceRepository(db)
	maintenanceRepo := postgres.NewMaintenanceRepository(db)
	fuelRepo := postgres.NewFuelRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	recoveryRepo := postgres.NewRecoveryCodeRepository(db)
	alertRepo := postgres.NewAlertRepository(db)
	periodRepo := postgres.NewPeriodClosureRepository(db)
	calcRunRepo := postgres.NewCalculationRunRepository(db)

	mailer, err := newMailer(cfg, log)
	if err != nil {
		return err
	}
	notifications := services.NewNotificationService(notificationRepo, mailer, audit, clock, log).WithAlerts(alertRepo)

	exportService := services.NewExportService(siteRepo, reportRepo, disputeRepo, incidentRepo, txManager, audit, clock, loc).
		WithRuns(calcRunRepo)

	sd := &seeder{
		log: log, clock: clock, loc: loc,
		siteRepo: siteRepo, userRepo: userRepo, reportRepo: reportRepo,
		incidentRepo: incidentRepo, disputeRepo: disputeRepo, acceptanceRepo: acceptanceRepo,
		ticketRepo: ticketRepo, maintenanceRepo: maintenanceRepo, notificationRepo: notificationRepo,

		users:       services.NewUserService(userRepo, siteRepo, hasher, txManager, audit, clock, nil, recoveryRepo, notifications),
		sites:       services.NewSiteService(siteRepo, txManager, audit),
		reports:     services.NewReportService(reportRepo, siteRepo, txManager, audit, clock, loc),
		incidents:   services.NewIncidentService(incidentRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
		corrections: services.NewCorrectionService(incidentCorrectionRepo, incidentRepo, txManager, audit, clock),
		disputes:    services.NewDisputeService(disputeRepo, reportRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
		tickets:     services.NewTicketService(ticketRepo, incidentRepo, siteRepo, userRepo, txManager, audit, clock, notifications, cfg.Mail.BaseURL),
		acceptance:  services.NewAcceptanceService(acceptanceRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL),
		maintenance: services.NewMaintenanceService(maintenanceRepo, siteRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL,
			domain.MaintenanceRules{
				NoticeHours: cfg.Figures.MaintenanceNoticeHours, MaxHours: cfg.Figures.MaintenanceMaxHours,
				MonthlyCapHours: cfg.Figures.MaintenanceMonthlyCapHours,
			}),
		fuel:    services.NewFuelService(fuelRepo, siteRepo, reportRepo, userRepo, txManager, audit, clock, notifications, loc, cfg.Mail.BaseURL, cfg.Fuel.LowHours),
		periods: services.NewPeriodService(periodRepo, exportService, txManager, audit, clock, loc, cfg.Periods.CloseGraceDays),
		runs:    exportService,
	}
	return sd.run(ctx)
}

type seeder struct {
	log   *slog.Logger
	clock ports.Clock
	loc   *time.Location

	siteRepo         ports.SiteRepository
	userRepo         ports.UserRepository
	reportRepo       ports.ReportRepository
	incidentRepo     ports.IncidentRepository
	disputeRepo      ports.DisputeRepository
	acceptanceRepo   ports.AcceptanceRepository
	ticketRepo       ports.TicketRepository
	maintenanceRepo  ports.MaintenanceRepository
	notificationRepo ports.NotificationStore

	users       *services.UserService
	sites       *services.SiteService
	reports     *services.ReportService
	incidents   *services.IncidentService
	corrections *services.CorrectionService
	disputes    *services.DisputeService
	tickets     *services.TicketService
	acceptance  *services.AcceptanceService
	maintenance *services.MaintenanceService
	fuel        *services.FuelService
	periods     *services.PeriodService
	runs        *services.ExportService
}

// allPermissions is every permission the domain defines. A console seed tool
// is trusted by nature, the same way create-admin already bypasses the normal
// invite flow; this just extends that trust to the other services. Do not
// reuse this for a two-sided workflow (acceptance, disputes, tickets) -- see
// the narrow actors below.
func allPermissions() map[domain.Permission]struct{} {
	return map[domain.Permission]struct{}{
		domain.PermUsersManage:        {},
		domain.PermSitesAll:           {},
		domain.PermSitesRead:          {},
		domain.PermSitesManage:        {},
		domain.PermReportsCreate:      {},
		domain.PermReportsRead:        {},
		domain.PermReportsReview:      {},
		domain.PermAuditRead:          {},
		domain.PermExportsCreate:      {},
		domain.PermNotificationsRead:  {},
		domain.PermIncidentsCreate:    {},
		domain.PermIncidentsRead:      {},
		domain.PermIncidentsManage:    {},
		domain.PermDisputesRead:       {},
		domain.PermDisputesRaise:      {},
		domain.PermDisputesManage:     {},
		domain.PermTicketsRead:        {},
		domain.PermTicketsRaise:       {},
		domain.PermTicketsManage:      {},
		domain.PermAlertsRead:         {},
		domain.PermFuelRead:           {},
		domain.PermFuelRecord:         {},
		domain.PermFuelManage:         {},
		domain.PermPeriodsManage:      {},
		domain.PermMaintenanceRead:    {},
		domain.PermMaintenanceRequest: {},
		domain.PermMaintenanceApprove: {},
		domain.PermAcceptanceRead:     {},
		domain.PermAcceptanceManage:   {},
		domain.PermAcceptanceReview:   {},
	}
}

func seedActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: allPermissions()}
}

// Dispute, ticket and acceptance resolve which side an actor speaks for from
// the actor's own permissions (PartyOf / TicketPartyOf / AcceptanceParty),
// manage-side permissions winning over raise/review ones. A seedActor holding
// every permission would always resolve to Coloplus, so each of these three
// workflows needs its own two narrow actors, never the broad one above.

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

func coloplusDisputeActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermDisputesManage: {}, domain.PermDisputesRead: {},
	}}
}

func airtelDisputeActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermDisputesRaise: {}, domain.PermDisputesRead: {},
	}}
}

func coloplusTicketActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermTicketsManage: {}, domain.PermTicketsRead: {},
	}}
}

func airtelTicketActor(userID string) domain.Actor {
	return domain.Actor{UserID: userID, AllSites: true, Permissions: map[domain.Permission]struct{}{
		domain.PermTicketsRaise: {}, domain.PermTicketsRead: {},
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

// addMonths is like addDays, for the calendar-month arithmetic a period
// closure needs: comfortably past months, safe for any "today" this seed
// happens to run on.
func addMonths(date time.Time, months int) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m+time.Month(months), d, 0, 0, 0, 0, time.UTC)
}

// localDayEnd is the exclusive end of the reporting day in loc (next day's
// local midnight) -- the same bound domain.DayBounds computes.
func localDayEnd(date time.Time, loc *time.Location) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// atLocalHour anchors a maintenance window to a fixed morning hour on date,
// in loc -- never "now plus N hours", which can cross local midnight and
// trip the one-window-one-day rule depending on what time of day this
// command happens to run.
func atLocalHour(date time.Time, hour int, loc *time.Location) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, hour, 0, 0, 0, loc)
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

// ensureSiteSuspended puts a site into the one SiteStatus the acceptance
// workflow never produces on its own.
func (sd *seeder) ensureSiteSuspended(ctx context.Context, actor domain.Actor, site domain.Site) error {
	if site.Status == domain.SiteSuspended {
		return nil
	}
	site.Status = domain.SiteSuspended
	_, err := sd.sites.Update(ctx, actor, site.ID, site)
	return err
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

// ensureUserSites makes sure an (possibly pre-existing) user is assigned to
// every one of siteIDs -- a user created before a site existed, in an
// earlier run of this command, would not otherwise be, and incident
// ownership is checked against this real assignment, not an actor's
// permissions.
func (sd *seeder) ensureUserSites(ctx context.Context, actor domain.Actor, userID string, siteIDs []string) error {
	current, err := sd.userRepo.SiteIDsForUser(ctx, userID)
	if err != nil {
		return err
	}
	have := make(map[string]struct{}, len(current))
	for _, id := range current {
		have[id] = struct{}{}
	}
	missing := false
	for _, id := range siteIDs {
		if _, ok := have[id]; !ok {
			missing = true
			break
		}
	}
	if !missing {
		return nil
	}
	_, err = sd.users.Update(ctx, actor, userID, services.UpdateUserInput{SiteIDs: &siteIDs})
	return err
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
	// dailyIntervals always includes a short observed outage (DECISIONS 051
	// requires a reason to submit one, and domain.ValidateGeneratorHours
	// requires generator runtime hours too), so every report built from it
	// needs both set, whatever the target status.
	outageGeneratorHours := 0.5
	created, err := sd.reports.CreateDraft(ctx, reporter, siteID, date, services.ReportInput{
		Notes: "Seeded sample entry.", Intervals: intervals, GeneratorHours: &outageGeneratorHours,
		OutageReason: "Seeded sample outage: short morning utility dip before the generator backup engaged.",
	})
	if err != nil {
		return domain.DailyReport{}, err
	}
	if targetStatus == domain.ReportDraft {
		return created, nil
	}
	submitted, err := sd.reports.Submit(ctx, reporter, created.ID, "Entered from the sample data")
	if err != nil {
		return domain.DailyReport{}, err
	}
	if targetStatus == domain.ReportSubmitted {
		return submitted, nil
	}
	if targetStatus == domain.ReportRejected {
		return sd.reports.Reject(ctx, supervisor, submitted.ID, "Seeded sample rejection: the morning outage interval needs a reason.")
	}
	return sd.reports.Approve(ctx, supervisor, submitted.ID, "Looks consistent with the site's telemetry.")
}

// ensureSupersededReport corrects an already-approved report and carries the
// correction all the way to approved, so the original flips to superseded --
// the one report state ensureReportDay cannot reach on its own. This dev
// database has been running across several real days, so "a day n days ago"
// can land on a report that already exists from genuine past use rather than
// from this run -- StartCorrection carries that report's own OutageReason
// onto the draft, and an older report from before it was required may have
// none. Rather than fail the whole seed over one demo, this skips it.
func (sd *seeder) ensureSupersededReport(ctx context.Context, corrector, reviewer domain.Actor, approvedID string) error {
	approved, err := sd.reportRepo.GetByID(ctx, approvedID, false)
	if err != nil {
		return err
	}
	if approved.Status == domain.ReportSuperseded {
		sd.log.Info("correction already seeded", "report_id", approvedID)
		return nil
	}
	if approved.OutageReason == "" {
		sd.log.Warn("skipping superseded-report demo: the report to correct has no outage reason (it predates this seed's fix, or DECISIONS 051)", "report_id", approvedID)
		return nil
	}
	draft, err := sd.reports.StartCorrection(ctx, corrector, approvedID, "Seeded sample correction: the generator interval ran a little later than first recorded.")
	if err != nil {
		return err
	}
	submitted, err := sd.reports.Submit(ctx, corrector, draft.ID, "")
	if err != nil {
		return err
	}
	_, err = sd.reports.Approve(ctx, reviewer, submitted.ID, "Correction checked against the site's telemetry.")
	return err
}

type incidentSpec struct {
	siteID                  string
	title                   string
	category                domain.IncidentCategory
	severity                domain.Severity
	impact                  domain.Impact
	startedAt               time.Time
	target                  domain.IncidentStatus
	publish                 bool
	reporter, reporter2     domain.Actor
	ownerID                 string
	supervisor, supervisor2 domain.Actor
}

// incidentRank orders IncidentStatus along its one-way workflow, so progress
// already made (including from an earlier, partially-failed run) can be told
// from work still to do. Incidents are append-only -- there is no deleting a
// stuck one and starting over -- so ensureIncidentAt below must be able to
// resume from wherever an existing incident already is.
var incidentRank = map[domain.IncidentStatus]int{
	domain.IncidentReported:     0,
	domain.IncidentAcknowledged: 1,
	domain.IncidentAssigned:     1,
	domain.IncidentInProgress:   2,
	domain.IncidentRestored:     3,
	domain.IncidentVerified:     4,
	domain.IncidentClosed:       5,
}

// ensureIncidentAt reports an incident, or resumes one already reported under
// this title, and carries it forward through assignment, work, restoration,
// verification and closure -- stopping at whichever state is asked for, so
// every reachable IncidentStatus has at least one example in the sample data.
func (sd *seeder) ensureIncidentAt(ctx context.Context, spec incidentSpec) error {
	existing, _, err := sd.incidentRepo.List(ctx, ports.IncidentFilter{AllSites: true, SiteID: spec.siteID, Limit: 50})
	if err != nil {
		return err
	}
	var id string
	status := domain.IncidentReported
	for _, inc := range existing {
		if inc.Title == spec.title {
			id, status = inc.ID, inc.Status
			break
		}
	}
	if id == "" {
		created, err := sd.incidents.Report(ctx, spec.reporter, services.IncidentInput{
			SiteID: spec.siteID, Category: spec.category, Severity: spec.severity, Impact: spec.impact,
			Title:       spec.title,
			Description: "Seeded sample incident.",
			StartedAt:   spec.startedAt,
		})
		if err != nil {
			return err
		}
		id = created.ID
	} else if incidentRank[status] >= incidentRank[spec.target] {
		sd.log.Info("incident already seeded", "site_id", spec.siteID, "title", spec.title, "status", status)
		return nil
	}
	now := spec.startedAt

	if spec.target == domain.IncidentReported {
		return nil
	}
	if spec.target == domain.IncidentAcknowledged {
		if incidentRank[status] < incidentRank[domain.IncidentAcknowledged] {
			if _, err := sd.incidents.Transition(ctx, spec.supervisor, id, domain.Transition{To: domain.IncidentAcknowledged}); err != nil {
				return err
			}
		}
		return nil
	}

	// Everything past "acknowledged" is assigned first.
	if incidentRank[status] < incidentRank[domain.IncidentAssigned] {
		if _, err := sd.incidents.Assign(ctx, spec.supervisor, id, services.AssignInput{
			OwnerID: spec.ownerID, NextAction: "Confirm fuel levels and monitor restoration.",
		}); err != nil {
			return err
		}
	}
	if spec.target == domain.IncidentAssigned {
		return nil
	}

	if incidentRank[status] < incidentRank[domain.IncidentInProgress] {
		if _, err := sd.incidents.Transition(ctx, spec.supervisor, id, domain.Transition{To: domain.IncidentInProgress}); err != nil {
			return err
		}
	}
	if spec.target == domain.IncidentInProgress {
		return nil
	}

	restoredAt := now.Add(90 * time.Minute)
	if incidentRank[status] < incidentRank[domain.IncidentRestored] {
		if _, err := sd.incidents.Transition(ctx, spec.reporter, id, domain.Transition{
			To: domain.IncidentRestored, At: &restoredAt,
		}); err != nil {
			return err
		}
	}
	if spec.target == domain.IncidentRestored {
		return nil
	}

	verifiedAt := restoredAt.Add(30 * time.Minute)
	if incidentRank[status] < incidentRank[domain.IncidentVerified] {
		if _, err := sd.incidents.Transition(ctx, spec.supervisor2, id, domain.Transition{
			To: domain.IncidentVerified, At: &verifiedAt, Note: "Confirmed restored on a follow-up site check.",
		}); err != nil {
			return err
		}
	}
	if spec.target == domain.IncidentVerified {
		return nil
	}

	// Publish, if asked, before closing: Publish refuses a closed incident
	// (an incident goes to Airtel while still open; closing is the last word,
	// not a trigger to tell them for the first time).
	if spec.publish {
		if _, err := sd.incidents.Publish(ctx, spec.supervisor2, id); err != nil {
			return err
		}
	}
	_, err = sd.incidents.Transition(ctx, spec.supervisor2, id, domain.Transition{
		To: domain.IncidentClosed, RootCause: "Generator auto-start relay failed to engage; replaced and load-tested.",
	})
	return err
}

// ensureIncidentCorrection requests and decides one correction request
// against an open incident, covering the given outcome (approved or
// rejected).
func (sd *seeder) ensureIncidentCorrection(
	ctx context.Context, requester, reviewer domain.Actor, incidentID string,
	field domain.CorrectionField, corrected time.Time, reason string, approve bool,
) error {
	existing, _, err := sd.corrections.List(ctx, reviewer, services.ListIncidentCorrectionsInput{IncidentID: incidentID, Limit: 50})
	if err != nil {
		return err
	}
	for _, c := range existing {
		if c.Reason == reason {
			sd.log.Info("incident correction already seeded", "incident_id", incidentID, "status", c.Status)
			return nil
		}
	}
	created, err := sd.corrections.Request(ctx, requester, services.RequestCorrectionInput{
		IncidentID: incidentID, Field: field, CorrectedValue: corrected, Reason: reason,
	})
	if err != nil {
		return err
	}
	if approve {
		_, err = sd.corrections.Approve(ctx, reviewer, created.ID)
		return err
	}
	_, err = sd.corrections.Reject(ctx, reviewer, created.ID, "Seeded sample rejection: the original time already matches the site visit log.")
	return err
}

// ensureDisputeAt raises a dispute and, depending on target, carries it
// through Coloplus's review and Airtel's response -- stopping at whichever
// status is asked for, so every reachable DisputeStatus has an example.
func (sd *seeder) ensureDisputeAt(
	ctx context.Context, coloplus, airtel domain.Actor, siteID, reportID, title string, target domain.DisputeStatus,
) error {
	existing, _, err := sd.disputeRepo.List(ctx, ports.DisputeFilter{AllSites: true, SiteID: siteID, Limit: 50})
	if err != nil {
		return err
	}
	for _, d := range existing {
		if d.Title == title {
			sd.log.Info("dispute already seeded", "site_id", siteID, "title", title, "status", d.Status)
			return nil
		}
	}
	created, err := sd.disputes.Raise(ctx, airtel, services.DisputeInput{
		SiteID: siteID, ReportID: reportID, Category: domain.DisputeWrongDuration,
		Title:       title,
		Description: "Seeded sample dispute.",
	})
	if err != nil {
		return err
	}
	id := created.ID
	if target == domain.DisputeOpen {
		return nil
	}

	if _, err := sd.disputes.Transition(ctx, coloplus, id, domain.DisputeStep{To: domain.DisputeUnderReview}); err != nil {
		return err
	}
	if target == domain.DisputeUnderReview {
		return nil
	}

	switch target {
	case domain.DisputeAwaitingEvidence:
		_, err := sd.disputes.Transition(ctx, coloplus, id, domain.DisputeStep{
			To: domain.DisputeAwaitingEvidence, Note: "Seeded: please share the site log for this interval.",
		})
		return err
	case domain.DisputeAgreedCorrection:
		_, err := sd.disputes.Transition(ctx, coloplus, id, domain.DisputeStep{
			To: domain.DisputeAgreedCorrection, Note: "Seeded: agreed, a corrected report will be filed.",
		})
		return err
	case domain.DisputeResolved, domain.DisputeClosed:
		if _, err := sd.disputes.Transition(ctx, coloplus, id, domain.DisputeStep{
			To: domain.DisputeResolved, Note: "Seeded: checked against the site's telemetry; the original figures stand.",
			Outcome: domain.OutcomeNotUpheld,
		}); err != nil {
			return err
		}
		if target == domain.DisputeResolved {
			return nil
		}
		_, err := sd.disputes.Transition(ctx, airtel, id, domain.DisputeStep{To: domain.DisputeClosed})
		return err
	}
	return nil
}

// ensureTicketAt raises a support ticket and, depending on target, carries it
// through Coloplus's workflow.
func (sd *seeder) ensureTicketAt(
	ctx context.Context, coloplus, airtel domain.Actor, siteID, subject string,
	category domain.TicketCategory, priority domain.TicketPriority, target domain.TicketStatus,
) error {
	existing, _, err := sd.ticketRepo.List(ctx, ports.TicketFilter{AllSites: true, SiteID: siteID, Limit: 50})
	if err != nil {
		return err
	}
	for _, t := range existing {
		if t.Subject == subject {
			sd.log.Info("ticket already seeded", "site_id", siteID, "subject", subject, "status", t.Status)
			return nil
		}
	}
	created, err := sd.tickets.Raise(ctx, airtel, services.TicketInput{
		SiteID: siteID, Category: category, Priority: priority, Subject: subject,
		Description: "Seeded sample ticket.", ContactEmail: "ops.contact@airtel.africa",
	})
	if err != nil {
		return err
	}
	if target == domain.TicketOpen {
		return nil
	}
	if _, err := sd.tickets.Transition(ctx, coloplus, created.ID, domain.TicketStep{To: domain.TicketInProgress}); err != nil {
		return err
	}
	if target == domain.TicketInProgress {
		return nil
	}
	_, err = sd.tickets.Transition(ctx, coloplus, created.ID, domain.TicketStep{
		To: domain.TicketResolved, Note: "Seeded: resolved on site visit.",
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

// ensureAcceptanceBuilt stops once Coloplus has declared the site built, the
// step before it submits for inspection.
func (sd *seeder) ensureAcceptanceBuilt(ctx context.Context, coloplus domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	if _, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Install the generator and battery bank.",
	}); err != nil {
		return err
	}
	_, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{To: domain.StageBuilt})
	return err
}

// ensureAcceptanceReadyForInspection stops once Coloplus has submitted the
// dossier and it is waiting on Airtel.
func (sd *seeder) ensureAcceptanceReadyForInspection(ctx context.Context, coloplus domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	if _, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Commission the solar and battery backup.",
	}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{To: domain.StageBuilt}); err != nil {
		return err
	}
	_, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{
		To: domain.StageReadyForInspection, Note: "Solar array and battery backup commissioned; test logs attached.",
	})
	return err
}

// ensureAcceptanceRejected walks Coloplus through to ready, then has Airtel
// reject it, the negative-path end state.
func (sd *seeder) ensureAcceptanceRejected(ctx context.Context, coloplus, airtel domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	if _, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Build out the generator enclosure.",
	}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{To: domain.StageBuilt}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{
		To: domain.StageReadyForInspection, Note: "Generator enclosure complete; photos attached.",
	}); err != nil {
		return err
	}
	if _, err := sd.acceptance.Transition(ctx, airtel, siteID, domain.AcceptanceStep{To: domain.StageInspected}); err != nil {
		return err
	}
	_, err := sd.acceptance.Transition(ctx, airtel, siteID, domain.AcceptanceStep{
		To: domain.StageRejected, Note: "Seeded: the enclosure does not meet the ventilation spec.",
	})
	return err
}

// ensureAcceptanceOnHold requests, then puts the process on hold.
func (sd *seeder) ensureAcceptanceOnHold(ctx context.Context, coloplus domain.Actor, siteID string) error {
	if _, err := sd.acceptanceRepo.GetBySite(ctx, siteID, false); err == nil {
		sd.log.Info("acceptance already seeded", "site_id", siteID)
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	if _, err := sd.acceptance.Request(ctx, coloplus, services.RequestAcceptanceInput{
		SiteID: siteID, NextAction: "Awaiting right-of-way approval before building out.",
	}); err != nil {
		return err
	}
	_, err := sd.acceptance.Transition(ctx, coloplus, siteID, domain.AcceptanceStep{
		To: domain.StageOnHold, Note: "Seeded: on hold pending the landlord's right-of-way approval.",
	})
	return err
}

func (sd *seeder) ensureMaintenanceWindow(
	ctx context.Context, requester, decider domain.Actor, siteID, reason string, start time.Time, target domain.MaintenanceStatus,
) error {
	existing, _, err := sd.maintenanceRepo.List(ctx, ports.MaintenanceFilter{AllSites: true, SiteID: siteID, Limit: 50})
	if err != nil {
		return err
	}
	for _, w := range existing {
		if w.Reason == reason {
			sd.log.Info("maintenance window already seeded", "site_id", siteID, "reason", reason, "status", w.Status)
			return nil
		}
	}
	created, err := sd.maintenance.Request(ctx, requester, siteID, services.MaintenanceInput{
		StartsAt: start, EndsAt: start.Add(2 * time.Hour), Reason: reason,
	})
	if err != nil {
		return err
	}
	switch target {
	case domain.MaintenanceRequested:
		return nil
	case domain.MaintenanceApproved:
		_, err = sd.maintenance.Approve(ctx, decider, created.ID, "Seeded: approved, notice and cap are both within range.")
	case domain.MaintenanceRejected:
		_, err = sd.maintenance.Reject(ctx, decider, created.ID, "Seeded: please combine this with next month's window instead.")
	case domain.MaintenanceCancelled:
		_, err = sd.maintenance.Cancel(ctx, requester, created.ID, "Seeded: no longer needed; the part arrived early.")
	}
	return err
}

// ensureFuelEntry relies on FuelService.Record's own client-key idempotency:
// a retried request with the same key returns the entry already made.
func (sd *seeder) ensureFuelEntry(ctx context.Context, actor domain.Actor, siteID, clientKey string, in services.FuelInput) (domain.FuelEntry, error) {
	in.ClientKey = clientKey
	recorded, err := sd.fuel.Record(ctx, actor, siteID, in)
	return recorded.Entry, err
}

func (sd *seeder) ensureFuelVoided(ctx context.Context, actor domain.Actor, entryID string) error {
	entry, err := sd.fuel.Void(ctx, actor, entryID, "Seeded sample void: reading entered against the wrong site, re-recorded correctly.")
	_ = entry
	return err
}

// ensureFailedNotification writes one notification straight through the
// outbox repository and marks it failed, the way the dispatcher itself would
// after Resend rejected it. The log mailer used in development always
// succeeds, so there is no way to reach this state through the ordinary
// enqueue-and-dispatch flow; this exists only so the admin email log's
// Failed tab has a realistic example to show and retry.
func (sd *seeder) ensureFailedNotification(ctx context.Context, actor domain.Actor, recipient, subject, dedupeKey, reason string) error {
	existing, _, err := sd.notificationRepo.List(ctx, ports.NotificationFilter{Status: domain.NotificationFailed, Limit: 200})
	if err != nil {
		return err
	}
	for _, n := range existing {
		if n.Subject == subject && n.Recipient == recipient {
			sd.log.Info("failed notification already seeded", "subject", subject)
			return nil
		}
	}
	now := sd.clock.Now()
	created, err := sd.notificationRepo.Enqueue(ctx, domain.Notification{
		Kind: "digest.weekly", Recipient: recipient, Subject: subject, Body: "Seeded sample email body.",
		DedupeKey: dedupeKey, NextAttemptAt: now,
	})
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	list, _, err := sd.notificationRepo.List(ctx, ports.NotificationFilter{Status: domain.NotificationPending, Limit: 200})
	if err != nil {
		return err
	}
	for _, n := range list {
		if n.DedupeKey == dedupeKey {
			return sd.notificationRepo.MarkFailed(ctx, n.ID, reason)
		}
	}
	return nil
}

// ensureCalculationRun stores a run only if one with this label does not
// already exist -- StoreRun itself has no dedupe, unlike almost everything
// else here.
func (sd *seeder) ensureCalculationRun(ctx context.Context, actor domain.Actor, month time.Time, label string) (domain.CalculationRun, error) {
	runs, _, err := sd.runs.Runs(ctx, actor, 50, 0)
	if err != nil {
		return domain.CalculationRun{}, err
	}
	for _, r := range runs {
		if r.Label == label {
			sd.log.Info("calculation run already seeded", "label", label)
			return r, nil
		}
	}
	return sd.runs.StoreRun(ctx, actor, month, "", label)
}

func (sd *seeder) ensurePeriodClosed(ctx context.Context, actor domain.Actor, month time.Time, reopen bool) error {
	closures, err := sd.periods.List(ctx, actor)
	if err != nil {
		return err
	}
	key := domain.MonthOf(month).Format("2006-01")
	for _, c := range closures {
		if c.Month.Format("2006-01") != key {
			continue
		}
		if reopen && c.Open() {
			_, err := sd.periods.Reopen(ctx, actor, month, "Seeded sample reopen: a late correction needed to go in after closing.")
			return err
		}
		sd.log.Info("period closure already seeded", "month", key, "reopened", !c.Open())
		return nil
	}
	if _, err := sd.periods.Close(ctx, actor, month); err != nil {
		return err
	}
	if !reopen {
		return nil
	}
	_, err = sd.periods.Reopen(ctx, actor, month, "Seeded sample reopen: a late correction needed to go in after closing.")
	return err
}

func (sd *seeder) run(ctx context.Context) error {
	bootstrap := seedActor("system-seed")

	lekkiRate, lekkiTank := 3.5, 500.0
	ikejaRate, ikejaTank := 2.8, 300.0
	// Real approximate coordinates for each named location, and a spread of
	// past commissioning dates -- flavour for the site profile fields that
	// have no computed default (GeneratorCapacity/AccessContact*/
	// SupervisorName/Latitude/Longitude/CommissioningDate), unlike Status/
	// ServiceWindow/FreshnessHours which SiteService.withDefaults already
	// fills in when left zero.
	lekkiLat, lekkiLng := 6.4474, 3.4737
	ikejaLat, ikejaLng := 6.5952, 3.3430
	garkiLat, garkiLng := 9.0379, 7.4951
	phcLat, phcLng := 4.7719, 7.0134
	kanoLat, kanoLng := 12.0022, 8.5411
	enuguLat, enuguLng := 6.4483, 7.5139
	lekkiCommissioned := time.Date(2022, time.March, 14, 0, 0, 0, 0, time.UTC)
	ikejaCommissioned := time.Date(2022, time.June, 2, 0, 0, 0, 0, time.UTC)
	garkiCommissioned := time.Date(2022, time.September, 20, 0, 0, 0, 0, time.UTC)
	phcCommissioned := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)
	kanoCommissioned := time.Date(2023, time.April, 30, 0, 0, 0, 0, time.UTC)
	enuguCommissioned := time.Date(2023, time.July, 8, 0, 0, 0, 0, time.UTC)
	siteSpecs := []domain.Site{
		{
			AirtelCode: "ATL-LEK-001", ColoplusCode: "CPL-LEK-001", Name: "Lekki Toll Gate Mast",
			Address: "Lekki-Epe Expressway, Toll Gate", Region: "South West", State: "Lagos", SiteType: "Mast",
			PowerSources:   []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator},
			ConsumptionLPH: &lekkiRate, TankCapacityLitres: &lekkiTank,
			ResponsibleTeam: "Lagos South Field Team", GeneratorCapacity: "60 kVA diesel generator",
			AccessContactName: "Tunde Bakare", AccessContactPhone: "+234 802 123 4567", SupervisorName: "Sam Supervisor",
			CommissioningDate: &lekkiCommissioned, Latitude: &lekkiLat, Longitude: &lekkiLng,
		},
		{
			AirtelCode: "ATL-IKJ-002", ColoplusCode: "CPL-IKJ-002", Name: "Ikeja Central Hub",
			Address: "Oba Akran Avenue, Ikeja", Region: "South West", State: "Lagos", SiteType: "Hub",
			PowerSources:   []domain.PowerSource{domain.SourceGrid, domain.SourceSolar},
			ConsumptionLPH: &ikejaRate, TankCapacityLitres: &ikejaTank,
			ResponsibleTeam: "Lagos North Field Team", GeneratorCapacity: "45 kVA diesel generator",
			AccessContactName: "Ngozi Eze", AccessContactPhone: "+234 803 234 5678", SupervisorName: "Sam Supervisor",
			CommissioningDate: &ikejaCommissioned, Latitude: &ikejaLat, Longitude: &ikejaLng,
		},
		{
			AirtelCode: "ATL-GRK-003", ColoplusCode: "CPL-GRK-003", Name: "Abuja Garki Node",
			Address: "Area 3, Garki", Region: "North Central", State: "FCT", SiteType: "Node",
			PowerSources: []domain.PowerSource{domain.SourceGenerator, domain.SourceBattery},
			ResponsibleTeam: "FCT Field Team", GeneratorCapacity: "30 kVA diesel generator",
			AccessContactName: "Musa Ibrahim", AccessContactPhone: "+234 805 345 6789", SupervisorName: "Tunde Supervisor",
			CommissioningDate: &garkiCommissioned, Latitude: &garkiLat, Longitude: &garkiLng,
			// The one site in this sample set with the strict "unknown"
			// reading -- unreported time here is neither up nor down,
			// unlike every other site's default "available" reading.
			UnreportedTime: domain.UnreportedUnknown,
		},
		{
			AirtelCode: "ATL-PHC-004", ColoplusCode: "CPL-PHC-004", Name: "Port Harcourt Trans-Amadi Site",
			Address: "Trans-Amadi Industrial Layout", Region: "South South", State: "Rivers", SiteType: "Tower",
			PowerSources: []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator, domain.SourceSolar},
			ResponsibleTeam: "South South Field Team", GeneratorCapacity: "50 kVA diesel generator",
			AccessContactName: "Blessing Okafor", AccessContactPhone: "+234 806 456 7890", SupervisorName: "Tunde Supervisor",
			CommissioningDate: &phcCommissioned, Latitude: &phcLat, Longitude: &phcLng,
		},
		{
			AirtelCode: "ATL-KAN-005", ColoplusCode: "CPL-KAN-005", Name: "Kano Sabon Gari Node",
			Address: "Sabon Gari Market Road", Region: "North West", State: "Kano", SiteType: "Node",
			PowerSources: []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator},
			ResponsibleTeam: "North West Field Team", GeneratorCapacity: "35 kVA diesel generator",
			AccessContactName: "Abubakar Sani", AccessContactPhone: "+234 807 567 8901", SupervisorName: "Sam Supervisor",
			CommissioningDate: &kanoCommissioned, Latitude: &kanoLat, Longitude: &kanoLng,
		},
		{
			AirtelCode: "ATL-ENU-006", ColoplusCode: "CPL-ENU-006", Name: "Enugu Independence Layout Hub",
			Address: "Independence Layout", Region: "South East", State: "Enugu", SiteType: "Hub",
			PowerSources: []domain.PowerSource{domain.SourceGenerator, domain.SourceSolar, domain.SourceBattery},
			ResponsibleTeam: "South East Field Team", GeneratorCapacity: "40 kVA diesel generator",
			AccessContactName: "Chukwuemeka Nwosu", AccessContactPhone: "+234 808 678 9012", SupervisorName: "Tunde Supervisor",
			CommissioningDate: &enuguCommissioned, Latitude: &enuguLat, Longitude: &enuguLng,
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
	lekkiID, ikejaID, garkiID, phcID, kanoID, enuguID := siteIDs[0], siteIDs[1], siteIDs[2], siteIDs[3], siteIDs[4], siteIDs[5]

	// Users: two per role. The first of each pair keeps the original email so
	// a database seeded before this change still matches by email and is left
	// alone, not duplicated.
	admin, err := sd.ensureUser(ctx, bootstrap, "admin@coloplus.africa", "Ada Admin", domain.RoleAdmin, nil)
	if err != nil {
		return fmt.Errorf("seeding admin: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "admin2@coloplus.africa", "Femi Admin", domain.RoleAdmin, nil); err != nil {
		return fmt.Errorf("seeding second admin: %w", err)
	}
	supervisor, err := sd.ensureUser(ctx, bootstrap, "supervisor@coloplus.africa", "Sam Supervisor", domain.RoleSupervisor, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding supervisor: %w", err)
	}
	supervisor2, err := sd.ensureUser(ctx, bootstrap, "supervisor2@coloplus.africa", "Tunde Supervisor", domain.RoleSupervisor, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding second supervisor: %w", err)
	}
	// Both supervisors own incidents at every site below, including the two
	// added in this change -- a supervisor found pre-existing from before
	// those sites existed needs its assignment widened to match.
	if err := sd.ensureUserSites(ctx, bootstrap, supervisor.ID, siteIDs); err != nil {
		return fmt.Errorf("widening supervisor's site assignment: %w", err)
	}
	if err := sd.ensureUserSites(ctx, bootstrap, supervisor2.ID, siteIDs); err != nil {
		return fmt.Errorf("widening second supervisor's site assignment: %w", err)
	}
	reporter, err := sd.ensureUser(ctx, bootstrap, "reporter@coloplus.africa", "Remi Reporter", domain.RoleReporter, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding reporter: %w", err)
	}
	reporter2, err := sd.ensureUser(ctx, bootstrap, "reporter2@coloplus.africa", "Bola Reporter", domain.RoleReporter, []string{lekkiID, ikejaID})
	if err != nil {
		return fmt.Errorf("seeding second reporter: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "viewer@airtel.africa", "Vera Viewer", domain.RoleAirtelViewer, siteIDs); err != nil {
		return fmt.Errorf("seeding airtel viewer: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "viewer2@airtel.africa", "Dayo Viewer", domain.RoleAirtelViewer, siteIDs); err != nil {
		return fmt.Errorf("seeding second airtel viewer: %w", err)
	}
	reviewer, err := sd.ensureUser(ctx, bootstrap, "reviewer@airtel.africa", "Rex Reviewer", domain.RoleAirtelReviewer, siteIDs)
	if err != nil {
		return fmt.Errorf("seeding airtel reviewer: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "reviewer2@airtel.africa", "Nkem Reviewer", domain.RoleAirtelReviewer, siteIDs); err != nil {
		return fmt.Errorf("seeding second airtel reviewer: %w", err)
	}
	// Auditor is site-scoped, like a reporter (DECISIONS 045) -- nil site ids
	// would leave both accounts able to sign in but seeing no sites, no
	// reports and no maintenance at all. The first gets every site (a
	// portfolio-wide auditor); the second gets a subset, the same
	// first-gets-everything/second-gets-a-subset spread reporter/reporter2
	// already demonstrate above.
	if _, err := sd.ensureUser(ctx, bootstrap, "auditor@coloplus.africa", "Amaka Auditor", domain.RoleAuditor, siteIDs); err != nil {
		return fmt.Errorf("seeding auditor: %w", err)
	}
	if _, err := sd.ensureUser(ctx, bootstrap, "auditor2@coloplus.africa", "Chidi Auditor", domain.RoleAuditor, []string{garkiID, phcID, kanoID}); err != nil {
		return fmt.Errorf("seeding second auditor: %w", err)
	}

	reporterActor := seedActor(reporter.ID)
	reporter2Actor := seedActor(reporter2.ID)
	supervisorActor := seedActor(supervisor.ID)
	supervisor2Actor := seedActor(supervisor2.ID)

	// Daily reports: the two older days approved on both Lagos sites; today is
	// left in progress -- submitted on Lekki, still a draft on Ikeja. A third
	// site's yesterday is submitted then rejected, and one of Lekki's approved
	// days is corrected end to end, so draft, submitted, approved, rejected
	// and superseded are all on record.
	now := sd.clock.Now()
	today := reportingDate(now, sd.loc)
	yesterday := addDays(today, -1)
	dayBefore := addDays(today, -2)
	endOfToday := now.Truncate(time.Minute)

	type reportSite struct {
		id              string
		primary, backup domain.PowerSource
		todayTarget     domain.ReportStatus
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
	garkiYesterday := dailyIntervals(yesterday, sd.loc, domain.SourceGenerator, domain.SourceBattery, localDayEnd(yesterday, sd.loc))
	if _, err := sd.ensureReportDay(ctx, reporter2Actor, supervisorActor, garkiID, yesterday, garkiYesterday, domain.ReportRejected); err != nil {
		return fmt.Errorf("seeding rejected report: %w", err)
	}

	// A dedicated day for the supersede demo, not shared with the loop above:
	// an older database may already hold dayBefore/yesterday's Lekki reports
	// from before OutageReason was required here, and StartCorrection carries
	// the original's (possibly still-empty) reason onto the correction draft.
	// A fresh day guarantees the correction has one to inherit.
	supersedeDay := addDays(today, -10)
	supersedeIntervals := dailyIntervals(supersedeDay, sd.loc, domain.SourceGrid, domain.SourceGenerator, localDayEnd(supersedeDay, sd.loc))
	supersedeSource, err := sd.ensureReportDay(ctx, reporterActor, supervisorActor, lekkiID, supersedeDay, supersedeIntervals, domain.ReportApproved)
	if err != nil {
		return fmt.Errorf("seeding report to supersede: %w", err)
	}
	if err := sd.ensureSupersededReport(ctx, reporter2Actor, supervisor2Actor, supersedeSource.ID); err != nil {
		return fmt.Errorf("seeding superseded report: %w", err)
	}

	// Incidents: one per reachable status, across the six sites.
	incidentSpecs := []incidentSpec{
		{
			siteID: lekkiID, title: "Grid outage at Lekki Toll Gate Mast", category: domain.CategoryGridLoss,
			severity: domain.SeverityMajor, impact: domain.ImpactOutage, startedAt: now.Add(-3 * time.Hour),
			target: domain.IncidentAssigned, ownerID: supervisor.ID,
			reporter: reporterActor, reporter2: reporter2Actor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: ikejaID, title: "Generator failure at Ikeja Central Hub", category: domain.CategoryGeneratorFailure,
			severity: domain.SeverityMinor, impact: domain.ImpactNoOutage, startedAt: now.Add(-6 * time.Hour),
			target: domain.IncidentReported, ownerID: supervisor.ID,
			reporter: reporter2Actor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: garkiID, title: "Low fuel warning at Abuja Garki Node", category: domain.CategoryLowFuel,
			severity: domain.SeverityMinor, impact: domain.ImpactNoOutage, startedAt: now.Add(-5 * time.Hour),
			target: domain.IncidentAcknowledged, ownerID: supervisor2.ID,
			reporter: reporterActor, reporter2: reporter2Actor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: phcID, title: "Battery fault at Port Harcourt Trans-Amadi Site", category: domain.CategoryBatteryFailure,
			severity: domain.SeverityMajor, impact: domain.ImpactOutage, startedAt: now.Add(-8 * time.Hour),
			target: domain.IncidentInProgress, ownerID: supervisor.ID,
			reporter: reporter2Actor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: kanoID, title: "Rectifier fault at Kano Sabon Gari Node", category: domain.CategoryRectifierFault,
			severity: domain.SeverityMajor, impact: domain.ImpactOutage, startedAt: now.Add(-20 * time.Hour),
			target: domain.IncidentRestored, ownerID: supervisor2.ID,
			reporter: reporterActor, reporter2: reporter2Actor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: enuguID, title: "Access restricted at Enugu Independence Layout Hub", category: domain.CategoryAccessRestriction,
			severity: domain.SeverityCritical, impact: domain.ImpactOutage, startedAt: now.Add(-30 * time.Hour),
			target: domain.IncidentVerified, ownerID: supervisor.ID,
			reporter: reporter2Actor, reporter2: reporterActor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
		{
			siteID: lekkiID, title: "Electrical fault at the Lekki Toll Gate Mast feeder panel", category: domain.CategoryElectricalFault,
			severity: domain.SeverityCritical, impact: domain.ImpactOutage, startedAt: now.Add(-48 * time.Hour),
			target: domain.IncidentClosed, publish: true, ownerID: supervisor2.ID,
			reporter: reporterActor, reporter2: reporter2Actor, supervisor: supervisorActor, supervisor2: supervisor2Actor,
		},
	}
	var verifiedIncidentID string
	for _, spec := range incidentSpecs {
		if err := sd.ensureIncidentAt(ctx, spec); err != nil {
			return fmt.Errorf("seeding incident %q: %w", spec.title, err)
		}
		if spec.target == domain.IncidentVerified {
			list, _, err := sd.incidentRepo.List(ctx, ports.IncidentFilter{AllSites: true, SiteID: spec.siteID, Limit: 50})
			if err == nil {
				for _, inc := range list {
					if inc.Title == spec.title {
						verifiedIncidentID = inc.ID
					}
				}
			}
		}
	}
	if verifiedIncidentID != "" {
		if err := sd.ensureIncidentCorrection(ctx, reporter2Actor, supervisorActor, verifiedIncidentID,
			domain.FieldStartedAt, now.Add(-31*time.Hour), "Seeded sample correction: the fault log shows it starting a little earlier.", true,
		); err != nil {
			return fmt.Errorf("seeding approved incident correction: %w", err)
		}
		if err := sd.ensureIncidentCorrection(ctx, reporterActor, supervisor2Actor, verifiedIncidentID,
			domain.FieldRestoredAt, now.Add(-27*time.Hour), "Seeded sample correction: restoration may have been a little later than logged.", false,
		); err != nil {
			return fmt.Errorf("seeding rejected incident correction: %w", err)
		}
	}

	// Disputes: Airtel's reviewer disputes reports and sites across the
	// workflow's six statuses.
	coloplusDispute, airtelDispute := coloplusDisputeActor(supervisor.ID), airtelDisputeActor(reviewer.ID)
	disputeSpecs := []struct {
		siteID, reportID, title string
		target                  domain.DisputeStatus
	}{
		{lekkiID, lekkiYesterdayReportID, "Reported outage duration looks short for this period", domain.DisputeOpen},
		{ikejaID, "", "Missing outage not reflected in the report", domain.DisputeUnderReview},
		{garkiID, "", "Served minutes look too high for this day", domain.DisputeAwaitingEvidence},
		{phcID, "", "Wrong metric recorded for this period", domain.DisputeAgreedCorrection},
		{kanoID, "", "Coverage figure disputed for this site", domain.DisputeResolved},
		{enuguID, "", "Outage duration already addressed", domain.DisputeClosed},
	}
	for _, spec := range disputeSpecs {
		if err := sd.ensureDisputeAt(ctx, coloplusDispute, airtelDispute, spec.siteID, spec.reportID, spec.title, spec.target); err != nil {
			return fmt.Errorf("seeding dispute %q: %w", spec.title, err)
		}
	}

	// Support tickets: Airtel raises, Coloplus works them.
	coloplusTicket, airtelTicket := coloplusTicketActor(supervisor2.ID), airtelTicketActor(reviewer.ID)
	ticketSpecs := []struct {
		siteID, subject string
		category        domain.TicketCategory
		priority        domain.TicketPriority
		target          domain.TicketStatus
	}{
		{lekkiID, "Generator making unusual noise", domain.TicketGeneratorFault, domain.TicketMedium, domain.TicketOpen},
		{ikejaID, "Tower light not working", domain.TicketEquipmentFault, domain.TicketLow, domain.TicketInProgress},
		{garkiID, "Fence damaged by weather", domain.TicketAccessRestriction, domain.TicketHigh, domain.TicketResolved},
	}
	for _, spec := range ticketSpecs {
		if err := sd.ensureTicketAt(ctx, coloplusTicket, airtelTicket, spec.siteID, spec.subject, spec.category, spec.priority, spec.target); err != nil {
			return fmt.Errorf("seeding ticket %q: %w", spec.subject, err)
		}
	}

	// Acceptance: the full spread across six sites -- accepted (activates the
	// site), requested, built, ready for inspection, rejected and on hold.
	coloplusAcceptance, airtelAcceptance := coloplusAcceptanceActor(supervisor.ID), airtelAcceptanceActor(reviewer.ID)
	if err := sd.ensureAcceptanceAccepted(ctx, coloplusAcceptance, airtelAcceptance, lekkiID); err != nil {
		return fmt.Errorf("seeding Lekki acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceRequested(ctx, coloplusAcceptance, ikejaID); err != nil {
		return fmt.Errorf("seeding Ikeja acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceBuilt(ctx, coloplusAcceptance, garkiID); err != nil {
		return fmt.Errorf("seeding Garki acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceRejected(ctx, coloplusAcceptance, airtelAcceptance, phcID); err != nil {
		return fmt.Errorf("seeding Port Harcourt acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceOnHold(ctx, coloplusAcceptance, kanoID); err != nil {
		return fmt.Errorf("seeding Kano acceptance: %w", err)
	}
	if err := sd.ensureAcceptanceReadyForInspection(ctx, coloplusAcceptance, enuguID); err != nil {
		return fmt.Errorf("seeding Enugu acceptance: %w", err)
	}
	// Enugu also demonstrates the one SiteStatus the acceptance workflow never
	// reaches on its own.
	enuguSite, err := sd.siteRepo.GetByID(ctx, enuguID)
	if err != nil {
		return fmt.Errorf("loading Enugu site: %w", err)
	}
	if err := sd.ensureSiteSuspended(ctx, bootstrap, enuguSite); err != nil {
		return fmt.Errorf("seeding Enugu suspension: %w", err)
	}

	// Planned maintenance at Lekki: requested, approved, rejected and
	// cancelled, each at least the notice period ahead.
	maintenanceSpecs := []struct {
		reason string
		start  time.Time
		target domain.MaintenanceStatus
	}{
		{"Transformer inspection", atLocalHour(addDays(today, 4), 9, sd.loc), domain.MaintenanceRequested},
		{"Generator service", atLocalHour(addDays(today, 5), 9, sd.loc), domain.MaintenanceApproved},
		{"Battery bank replacement", atLocalHour(addDays(today, 6), 9, sd.loc), domain.MaintenanceRejected},
		{"Antenna realignment", atLocalHour(addDays(today, 7), 9, sd.loc), domain.MaintenanceCancelled},
	}
	for _, spec := range maintenanceSpecs {
		if err := sd.ensureMaintenanceWindow(ctx, reporterActor, supervisorActor, lekkiID, spec.reason, spec.start, spec.target); err != nil {
			return fmt.Errorf("seeding maintenance %q: %w", spec.reason, err)
		}
	}

	// Fuel: Lekki gets a delivery, a tank reading (so a variance can be worked
	// out) and a voided mistaken entry; Ikeja gets a single delivery. The other
	// four sites have no agreed rate, so their estimate stays unavailable --
	// a real, and common, state in its own right.
	if _, err := sd.ensureFuelEntry(ctx, reporterActor, lekkiID, "seed-fuel-lekki-delivery-1", services.FuelInput{
		Kind: domain.FuelDelivery, Litres: 400, OccurredAt: addDays(today, -5), Note: "Seeded sample delivery.",
	}); err != nil {
		return fmt.Errorf("seeding Lekki fuel delivery: %w", err)
	}
	if _, err := sd.ensureFuelEntry(ctx, reporterActor, lekkiID, "seed-fuel-lekki-reading-1", services.FuelInput{
		Kind: domain.FuelReading, Litres: 340, OccurredAt: addDays(today, -1), Note: "Seeded sample tank reading.",
	}); err != nil {
		return fmt.Errorf("seeding Lekki fuel reading: %w", err)
	}
	mistaken, err := sd.ensureFuelEntry(ctx, reporterActor, lekkiID, "seed-fuel-lekki-mistaken-1", services.FuelInput{
		Kind: domain.FuelDelivery, Litres: 50, OccurredAt: addDays(today, -1), Note: "Seeded entry recorded against the wrong site.",
	})
	if err != nil {
		return fmt.Errorf("seeding Lekki mistaken fuel entry: %w", err)
	}
	if !mistaken.Voided() {
		if err := sd.ensureFuelVoided(ctx, reporterActor, mistaken.ID); err != nil {
			return fmt.Errorf("voiding Lekki fuel entry: %w", err)
		}
	}
	if _, err := sd.ensureFuelEntry(ctx, reporter2Actor, ikejaID, "seed-fuel-ikeja-delivery-1", services.FuelInput{
		Kind: domain.FuelDelivery, Litres: 250, OccurredAt: addDays(today, -3), Note: "Seeded sample delivery.",
	}); err != nil {
		return fmt.Errorf("seeding Ikeja fuel delivery: %w", err)
	}

	// One synthetic failed email, for the admin email log's Failed tab.
	if err := sd.ensureFailedNotification(ctx, bootstrap, "seed-unreachable@coloplus.example",
		"Weekly summary", "seed-failed-notification-1",
		"the service is temporarily unavailable, please try again shortly",
	); err != nil {
		return fmt.Errorf("seeding failed notification: %w", err)
	}

	// Calculation runs and period closures: an ad-hoc run on the current
	// month, a month closed two months ago (left closed) and a month closed
	// three months ago then reopened.
	// calculation_runs.created_by and period_closures.closed_by/reopened_by
	// are real foreign keys to users(id) -- the synthetic "system-seed"
	// bootstrap actor works for audit_log (a plain text column) but not here,
	// so these two use a real seeded admin instead.
	adminActor := seedActor(admin.ID)
	if _, err := sd.ensureCalculationRun(ctx, adminActor, reportingDate(now, sd.loc), "Seeded sample run"); err != nil {
		return fmt.Errorf("seeding calculation run: %w", err)
	}
	if err := sd.ensurePeriodClosed(ctx, adminActor, addMonths(today, -2), false); err != nil {
		return fmt.Errorf("seeding closed period: %w", err)
	}
	if err := sd.ensurePeriodClosed(ctx, adminActor, addMonths(today, -3), true); err != nil {
		return fmt.Errorf("seeding reopened period: %w", err)
	}

	sd.log.Info("seed complete -- sample logins (dev only), password for all",
		"password", seedPassword,
		"admin", "admin@coloplus.africa / admin2@coloplus.africa",
		"supervisor", "supervisor@coloplus.africa / supervisor2@coloplus.africa",
		"reporter", "reporter@coloplus.africa / reporter2@coloplus.africa",
		"airtel_viewer", "viewer@airtel.africa / viewer2@airtel.africa",
		"airtel_reviewer", "reviewer@airtel.africa / reviewer2@airtel.africa",
		"auditor", "auditor@coloplus.africa / auditor2@coloplus.africa",
	)
	return nil
}
