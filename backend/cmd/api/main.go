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
	"syscall"
	"time"

	"coloplus-backend/config"
	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/platform"
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
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q (known: migrate)", args[0])
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
	// Reserved for provider fan-out (email sends, exports) as features land.
	_ = workers

	tokenMaker, err := token.NewPasetoMaker(cfg.Auth.TokenKey)
	if err != nil {
		return fmt.Errorf("creating token maker: %w", err)
	}

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
	audit := services.NewAuditService(auditStore, auditBatcher, platform.SystemClock{}, log)

	// Scheduled jobs register here; a bad cron spec fails the boot.
	scheduler := jobs.NewScheduler(log)
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
		RateLimitCtx:   rateLimitCtx,
		SwaggerEnabled: cfg.Swagger.Enabled,
		TokenMaker:     tokenMaker,
		CookieName:     cfg.Auth.CookieName,
		Health:         services.NewHealthService(db),
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
