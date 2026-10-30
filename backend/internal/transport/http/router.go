// Package httpapi is the HTTP transport: parse, call a service, map the result.
// It holds no business logic.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"coloplus-backend/internal/dto"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/services"

	"pkg-common/middleware"
	"pkg-common/token"

	"github.com/gin-gonic/gin"
)

// Deps is everything the router needs, built once in main.
type Deps struct {
	Log        *slog.Logger
	Production bool

	CORSOrigins    []string
	RateLimit      middleware.RateLimitConfig
	AuthRateLimit  middleware.RateLimitConfig
	RateLimitCtx   context.Context
	SwaggerEnabled bool

	TokenMaker token.Maker
	Cookie     CookieConfig

	// Loc is the reporting time zone; Now supplies the current time.
	Loc *time.Location
	Now func() time.Time

	Health        *services.HealthService
	Auth          *services.AuthService
	Accounts      *services.AccountService
	Notifications *services.NotificationService
	Users         *services.UserService
	Sites         *services.SiteService
	Reports       *services.ReportService
	Incidents     *services.IncidentService
	Disputes      *services.DisputeService
	Audit         *services.AuditService
	Acceptance    *services.AcceptanceService
	SiteOverviews *services.SiteOverviewService
	Exports       *services.ExportService
	Dashboard     *services.DashboardService
}

// api carries what every handler shares.
type api struct {
	log           *slog.Logger
	cookie        CookieConfig
	loc           *time.Location
	clock         clock
	auth          *services.AuthService
	accounts      *services.AccountService
	notifications *services.NotificationService
	users         *services.UserService
	sites         *services.SiteService
	reports       *services.ReportService
	incidents     *services.IncidentService
	disputes      *services.DisputeService
	audit         *services.AuditService
	acceptance    *services.AcceptanceService
	siteOverviews *services.SiteOverviewService
	exports       *services.ExportService
	dashboard     *services.DashboardService
}

type clock struct{ now func() time.Time }

func (c clock) Now() time.Time { return c.now() }

func (a *api) fail(c *gin.Context, err error) { respondError(c, a.log, err) }

// NewRouter assembles the engine. Middleware order: request id, recover,
// CORS, security headers, rate limit, then auth on the groups that need it.
func NewRouter(d Deps) *gin.Engine {
	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	_ = r.SetTrustedProxies(nil)

	// Health sits outside rate limiting and logging: uptime monitors probe it
	// with GET and HEAD every minute.
	health := healthHandler{health: d.Health, log: d.Log}
	r.GET("/healthcheck", health.check)
	r.HEAD("/healthcheck", health.check)

	r.Use(
		requestID(),
		middleware.Recover(d.Log),
		corsMiddleware(d.CORSOrigins),
		securityHeaders(d.Production),
		middleware.RateLimit(d.RateLimitCtx, d.RateLimit),
		accessLog(d.Log),
	)

	if d.SwaggerEnabled {
		registerSwagger(r)
	}

	h := &api{
		log: d.Log, cookie: d.Cookie, loc: d.Loc, clock: clock{d.Now},
		auth: d.Auth, accounts: d.Accounts, notifications: d.Notifications, users: d.Users, sites: d.Sites, reports: d.Reports, incidents: d.Incidents, disputes: d.Disputes, audit: d.Audit, acceptance: d.Acceptance, siteOverviews: d.SiteOverviews, exports: d.Exports, dashboard: d.Dashboard,
	}

	v1 := r.Group("/api/v1", limitBody(), jsonWrites())
	v1.GET("/ping", ping)

	// Sign-in has its own, tighter limit per client IP.
	v1.POST("/auth/login", middleware.RateLimit(d.RateLimitCtx, d.AuthRateLimit), h.login)
	v1.POST("/auth/logout", h.logout)
	// Recovery and invitation links are reached without a session, so they get
	// the same tight per-IP limit as sign-in.
	v1.POST("/auth/forgot-password", middleware.RateLimit(d.RateLimitCtx, d.AuthRateLimit), h.forgotPassword)
	v1.POST("/auth/set-password", middleware.RateLimit(d.RateLimitCtx, d.AuthRateLimit), h.setPassword)

	// Everything below requires a valid token for an active user. Services
	// decide what each person may do, by permission and by site scope.
	authed := v1.Group("",
		middleware.Auth(d.TokenMaker, middleware.AuthConfig{CookieName: d.Cookie.Name}),
		loadActor(d.Auth, d.Log),
	)
	authed.GET("/auth/me", h.me)
	authed.POST("/auth/mfa/setup", h.setupMFA)
	authed.POST("/auth/mfa/enable", h.enableMFA)
	authed.POST("/auth/mfa/recovery-codes", h.newRecoveryCodes)

	authed.POST("/users", h.createUser)
	authed.GET("/users", h.listUsers)
	authed.GET("/users/:id", h.getUser)
	authed.PATCH("/users/:id", h.updateUser)
	authed.POST("/users/:id/invite", h.resendInvite)
	authed.POST("/users/:id/reset-mfa", h.resetUserMFA)

	authed.GET("/notifications", h.listNotifications)
	authed.POST("/notifications/:id/retry", h.retryNotification)

	authed.POST("/sites", h.createSite)
	authed.GET("/sites", h.listSites)
	authed.GET("/sites/:id", h.getSite)
	authed.GET("/sites/:id/overview", h.siteOverview)
	authed.PUT("/sites/:id", h.updateSite)
	authed.POST("/sites/:id/daily-reports", h.createReport)

	authed.GET("/daily-reports", h.listReports)
	authed.GET("/daily-reports/:id", h.getReport)
	authed.PUT("/daily-reports/:id", h.saveReport)
	authed.POST("/daily-reports/:id/submit", h.submitReport)
	authed.POST("/daily-reports/:id/approve", h.approveReport)
	authed.POST("/daily-reports/:id/reject", h.rejectReport)
	authed.POST("/daily-reports/:id/corrections", h.startCorrection)

	authed.POST("/sites/:id/incidents", h.reportIncident)
	authed.GET("/incidents", h.listIncidents)
	authed.GET("/incidents/summary", h.incidentSummary)
	authed.GET("/incidents/:id", h.getIncident)
	authed.POST("/incidents/:id/assign", h.assignIncident)
	authed.POST("/incidents/:id/transition", h.transitionIncident)
	authed.POST("/incidents/:id/updates", h.addIncidentUpdate)
	authed.POST("/incidents/:id/publish", h.publishIncident)

	authed.POST("/sites/:id/disputes", h.raiseDispute)
	authed.GET("/disputes", h.listDisputes)
	authed.GET("/disputes/summary", h.disputeSummary)
	authed.GET("/disputes/:id", h.getDispute)
	authed.POST("/disputes/:id/comments", h.commentDispute)
	authed.POST("/disputes/:id/transition", h.transitionDispute)

	authed.POST("/sites/:id/acceptance", h.requestAcceptance)
	authed.GET("/sites/:id/acceptance", h.getAcceptance)
	authed.POST("/sites/:id/acceptance/transition", h.transitionAcceptance)
	authed.GET("/acceptances", h.listAcceptances)

	authed.GET("/exports/daily-summaries.csv", h.exportDailySummaries)

	authed.GET("/audit", h.searchAudit)

	authed.GET("/dashboard/overview", h.overview)

	r.NoRoute(func(c *gin.Context) {
		respondError(c, d.Log, apperrors.ErrNotFound)
	})
	return r
}

// ping godoc
//
//	@Summary	API ping
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	dto.PingResponse
//	@Router		/api/v1/ping [get]
func ping(c *gin.Context) {
	c.JSON(http.StatusOK, dto.PingResponse{Message: "pong"})
}
