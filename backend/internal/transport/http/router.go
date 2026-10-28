// Package httpapi is the HTTP transport: parse, call a service, map the result.
// It holds no business logic.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

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
	RateLimitCtx   context.Context
	SwaggerEnabled bool

	TokenMaker token.Maker
	CookieName string

	Health *services.HealthService
}

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

	v1 := r.Group("/api/v1")
	v1.GET("/ping", ping)

	// Everything registered on authed requires a valid token. Feature routes
	// go here and are gated by permission, not role name.
	_ = v1.Group("", middleware.Auth(d.TokenMaker, middleware.AuthConfig{CookieName: d.CookieName}))

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
