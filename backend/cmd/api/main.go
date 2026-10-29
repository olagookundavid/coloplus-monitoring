// Command api is the Coloplus backend. This file is the composition root:
// it reads config, builds every concrete type, wires them together and owns
// the shutdown order. No business logic lives here.
//
//	@title			Coloplus Remote Monitoring API
//	@version		1.0
//	@description	Backend for the Coloplus remote monitoring portal.
//	@BasePath		/
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"coloplus-backend/config"
	"coloplus-backend/internal/adapters/mail"
	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/adapters/security"
	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/ports"
	"coloplus-backend/internal/services"
	"coloplus-backend/internal/sql/migrations"
	httpapi "coloplus-backend/internal/transport/http"
	"coloplus-backend/internal/transport/jobs"

	"pkg-common/logger"
	"pkg-common/middleware"
	"pkg-common/token"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	log := logger.New(logger.Options{
		App:         cfg.App.Name,
		Environment: cfg.App.Env,
		JSON:        cfg.IsProduction(),
		Level:       parseLevel(cfg.App.LogLevel),
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// `api migrate` applies pending migrations and exits. It runs as its own
	// step before a deploy, never implicitly on start.
	if len(args) > 0 && args[0] == "migrate" {
		return postgres.Migrate(ctx, cfg.DB.URL, migrations.FS, log)
	}
	// `api create-admin <email> <full name>` creates the first administrator.
	// The password is read from ADMIN_PASSWORD so it never lands in shell
	// history or the process list.
	if len(args) > 0 && args[0] == "create-admin" {
		return createAdmin(ctx, cfg, log, args[1:])
	}
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q (known: migrate, create-admin)", args[0])
	}
	return serve(ctx, cfg, log)
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	// Infrastructure.
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		URL:      cfg.DB.URL,
		MaxConns: cfg.DB.MaxConns,
		MinConns: cfg.DB.MinConns,
		Timezone: cfg.DB.Timezone,
	})
	if err != nil {
		return err
	}
	db := postgres.NewDB(pool)

	workers, err := platform.NewWorkerPool(platform.WorkerPoolConfig{
		Size:      cfg.Workers.Size,
		QueueSize: cfg.Workers.QueueSize,
	}, log)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(cfg.App.ReportingTimezone)
	if err != nil {
		return fmt.Errorf("loading reporting time zone: %w", err)
	}
	tokenMaker, err := token.NewPasetoMaker(cfg.Auth.TokenKey)
	if err != nil {
		return fmt.Errorf("creating token maker: %w", err)
	}
	secretBox, err := security.NewSecretBox(cfg.Auth.SecretsKey)
	if err != nil {
		return err
	}
	clock := platform.SystemClock{}
	hasher := security.NewBcryptHasher(cfg.Auth.BcryptCost)

	// Persistence.
	txManager := postgres.NewTxManager(db)
	userRepo := postgres.NewUserRepository(db)
	siteRepo := postgres.NewSiteRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	incidentRepo := postgres.NewIncidentRepository(db)
	disputeRepo := postgres.NewDisputeRepository(db)
	acceptanceRepo := postgres.NewAcceptanceRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	userTokenRepo := postgres.NewUserTokenRepository(db)

	// Audit: batched by default, transactional where a feature asks for it.
	auditStore := postgres.NewAuditRepository(db)
	auditBatcher, err := platform.NewBatcher(platform.BatcherConfig{
		Name:         "audit",
		MaxItems:     cfg.Audit.BatchSize,
		Interval:     cfg.Audit.BatchInterval,
		QueueSize:    cfg.Audit.QueueSize,
		Overflow:     platform.OverflowBlock,
		BlockTimeout: 250 * time.Millisecond,
	}, auditStore.InsertBatch, log)
	if err != nil {
		return err
	}
	audit := services.NewAuditService(auditStore, auditBatcher, clock, log)

	mailer, err := newMailer(cfg, log)
	if err != nil {
		return err
	}

	// Use cases.
	notifications := services.NewNotificationService(notificationRepo, mailer, audit, clock, log)
	accounts := services.NewAccountService(userRepo, userTokenRepo, hasher, txManager, notifications, audit, clock, cfg.Mail.BaseURL)
	incidents := services.NewIncidentService(incidentRepo, siteRepo, userRepo, txManager, audit, clock,
		notifications, loc, cfg.Mail.BaseURL)
	disputes := services.NewDisputeService(disputeRepo, reportRepo, siteRepo, userRepo, txManager, audit, clock,
		notifications, loc, cfg.Mail.BaseURL)
	acceptance := services.NewAcceptanceService(acceptanceRepo, siteRepo, userRepo, txManager, audit, clock,
		notifications, loc, cfg.Mail.BaseURL)
	reportAlerts := services.NewReportAlertService(siteRepo, reportRepo, userRepo, notifications, clock, loc, cfg.Mail.BaseURL)
	authService, err := services.NewAuthService(userRepo, hasher, security.NewPasetoIssuer(tokenMaker),
		security.NewTOTP(cfg.Auth.MFAIssuer), secretBox, audit, clock, cfg.Auth.AccessTokenTTL, log)
	if err != nil {
		return err
	}

	// Scheduled jobs register here; a bad cron spec fails the boot.
	scheduler := jobs.NewScheduler(log)
	if err := scheduler.Add("dispatch-emails", cfg.Jobs.DispatchEmailsCron, func(ctx context.Context) error {
		res, err := notifications.Dispatch(ctx)
		if res.Sent+res.Retrying+res.Failed > 0 {
			log.Info("emails dispatched", "sent", res.Sent, "retrying", res.Retrying, "failed", res.Failed)
		}
		return err
	}); err != nil {
		return err
	}
	if err := scheduler.Add("incident-alerts", cfg.Jobs.IncidentAlertsCron, func(ctx context.Context) error {
		n, err := incidents.AlertOverdue(ctx)
		if n > 0 {
			log.Info("overdue incident alerts queued", "emails", n)
		}
		return err
	}); err != nil {
		return err
	}
	if err := scheduler.Add("report-alerts", cfg.Jobs.ReportAlertsCron, func(ctx context.Context) error {
		n, err := reportAlerts.AlertOverdue(ctx)
		if n > 0 {
			log.Info("overdue report alerts queued", "emails", n)
		}
		return err
	}); err != nil {
		return err
	}
	if err := scheduler.Add("acceptance-alerts", cfg.Jobs.AcceptanceAlertsCron, func(ctx context.Context) error {
		n, err := acceptance.AlertOverdue(ctx)
		if n > 0 {
			log.Info("overdue acceptance alerts queued", "emails", n)
		}
		return err
	}); err != nil {
		return err
	}
	scheduler.Start()

	// Transport.
	rateLimitCtx, stopRateLimit := context.WithCancel(context.Background())
	router := httpapi.NewRouter(httpapi.Deps{
		Log:         log,
		Production:  cfg.IsProduction(),
		CORSOrigins: cfg.CORS.AllowOrigins,
		RateLimit: middleware.RateLimitConfig{
			Enabled: cfg.RateLimit.Enabled,
			Rps:     cfg.RateLimit.Rps,
			Burst:   cfg.RateLimit.Burst,
		},
		AuthRateLimit: middleware.RateLimitConfig{
			Enabled: cfg.RateLimit.Enabled,
			Rps:     float64(cfg.RateLimit.AuthPerMinute) / 60,
			Burst:   cfg.RateLimit.AuthBurst,
		},
		RateLimitCtx:   rateLimitCtx,
		SwaggerEnabled: cfg.Swagger.Enabled,
		TokenMaker:     tokenMaker,
		Cookie: httpapi.CookieConfig{
			Name: cfg.Auth.CookieName, Domain: cfg.Auth.CookieDomain,
			Secure: cfg.Auth.CookieSecure, SameSite: sameSite(cfg.Auth.CookieSameSite),
		},
		Loc:           loc,
		Now:           clock.Now,
		Health:        services.NewHealthService(db),
		Auth:          authService,
		Accounts:      accounts,
		Notifications: notifications,
		Users:         services.NewUserService(userRepo, siteRepo, hasher, txManager, audit, clock, accounts),
		Sites:         services.NewSiteService(siteRepo, txManager, audit),
		Reports:       services.NewReportService(reportRepo, siteRepo, txManager, audit, clock, loc),
		Incidents:     incidents,
		Disputes:      disputes,
		Audit:         audit,
		Acceptance:    acceptance,
		SiteOverviews: services.NewSiteOverviewService(siteRepo, reportRepo, incidentRepo, disputeRepo, acceptanceRepo, clock, loc),
		Dashboard:     services.NewDashboardService(siteRepo, reportRepo, clock, loc),
	})
	srv := &http.Server{
		Addr:              cfg.App.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	_ = audit.Record(ctx, domain.AuditEvent{Action: "system.started", EntityType: "system", EntityID: cfg.App.Name})

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.App.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		log.Error("server failed", "error", err)
	case <-ctx.Done():
		log.Info("shutting down")
	}

	return shutdown(cfg.App.ShutdownTimeout, log, srv, stopRateLimit, scheduler, audit, auditBatcher, workers, pool.Close)
}

// shutdown runs the fixed order: stop taking requests, stop the scheduler and
// cancel running jobs, flush batchers and drain workers, close the database.
func shutdown(
	timeout time.Duration,
	log *slog.Logger,
	srv *http.Server,
	stopRateLimit context.CancelFunc,
	scheduler *jobs.Scheduler,
	audit *services.AuditService,
	auditBatcher *platform.Batcher[domain.AuditEvent],
	workers *platform.WorkerPool,
	closeDB func(),
) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var errs []error

	if err := srv.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("http shutdown: %w", err))
	}
	stopRateLimit()

	scheduler.Stop()

	_ = audit.Record(ctx, domain.AuditEvent{Action: "system.stopping", EntityType: "system"})
	if err := auditBatcher.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("flushing audit batcher: %w", err))
	}
	if err := workers.Drain(ctx); err != nil {
		errs = append(errs, err)
	}

	closeDB()
	log.Info("shutdown complete", "audit_dropped", auditBatcher.Dropped(), "audit_failed", auditBatcher.Failed())
	return errors.Join(errs...)
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return l
}

func sameSite(v string) http.SameSite {
	switch strings.ToLower(v) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func createAdmin(ctx context.Context, cfg config.Config, log *slog.Logger, args []string) error {
	password := os.Getenv("ADMIN_PASSWORD")
	if len(args) < 2 || password == "" {
		return errors.New("usage: ADMIN_PASSWORD=... api create-admin <email> <full name>")
	}
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		URL: cfg.DB.URL, MaxConns: 2, MinConns: 0, Timezone: cfg.DB.Timezone,
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	db := postgres.NewDB(pool)
	auditStore := postgres.NewAuditRepository(db)
	// Audit rows are written inside the transaction, so no batcher is needed.
	audit := services.NewAuditService(auditStore, discardQueue{}, platform.SystemClock{}, log)
	users := services.NewUserService(postgres.NewUserRepository(db), postgres.NewSiteRepository(db),
		security.NewBcryptHasher(cfg.Auth.BcryptCost), postgres.NewTxManager(db), audit, platform.SystemClock{}, nil)

	admin, err := users.CreateInitialAdmin(ctx, args[0], strings.Join(args[1:], " "), password)
	if err != nil {
		return err
	}
	log.Info("administrator created", "email", admin.Email, "id", admin.ID)
	return nil
}

// discardQueue satisfies ports.AuditQueue for commands that only write
// transactional audit rows.
type discardQueue struct{}

func (discardQueue) Add(domain.AuditEvent) bool { return false }

// newMailer picks the email provider named in the config.
func newMailer(cfg config.Config, log *slog.Logger) (ports.Mailer, error) {
	if cfg.Mail.Provider == "resend" {
		return mail.NewResend(mail.ResendConfig{
			APIKey: cfg.Mail.ResendAPIKey, From: cfg.Mail.From,
			RatePerSecond: cfg.Mail.RatePerSecond, Timeout: cfg.Mail.Timeout,
		})
	}
	log.Warn("emails are only written to the log (MAIL_PROVIDER=log)")
	return mail.NewLog(log), nil
}
