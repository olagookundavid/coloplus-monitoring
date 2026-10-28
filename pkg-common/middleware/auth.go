// Package middleware holds the gin middleware shared by Fusewall services.
// Nothing in it may name a product: cookie names and limits come from config.
package middleware

import (
	"net/http"
	"strings"

	"pkg-common/token"

	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// AuthConfig tells the auth middleware where to look for the token.
type AuthConfig struct {
	// CookieName is the cookie that carries the access token for browsers.
	// Required: there is deliberately no default, so no product name leaks in.
	CookieName string
	// AllowQueryToken accepts ?token= for clients that cannot set headers
	// (for example WebSockets). Off by default because URLs end up in logs.
	AllowQueryToken bool
}

// Auth rejects requests that carry no valid token.
func Auth(maker token.Maker, cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c, cfg)
		if raw == "" {
			abort(c, http.StatusUnauthorized, "access token is missing")
			return
		}
		payload, err := maker.VerifyToken(raw)
		if err != nil {
			abort(c, http.StatusUnauthorized, err.Error())
			return
		}
		c.Set(authorizationPayloadKey, payload)
		c.Next()
	}
}

// OptionalAuth attributes the caller when a valid token is present and lets
// anonymous callers through otherwise.
func OptionalAuth(maker token.Maker, cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if raw := extractToken(c, cfg); raw != "" {
			if payload, err := maker.VerifyToken(raw); err == nil {
				c.Set(authorizationPayloadKey, payload)
			}
		}
		c.Next()
	}
}

// PayloadFrom returns the verified token payload set by Auth or OptionalAuth.
func PayloadFrom(c *gin.Context) (*token.Payload, bool) {
	v, ok := c.Get(authorizationPayloadKey)
	if !ok {
		return nil, false
	}
	p, ok := v.(*token.Payload)
	return p, ok
}

func extractToken(c *gin.Context, cfg AuthConfig) string {
	if cfg.CookieName != "" {
		if cookie, err := c.Request.Cookie(cfg.CookieName); err == nil && cookie.Value != "" {
			return cookie.Value
		}
	}
	if fields := strings.Fields(c.GetHeader(authorizationHeaderKey)); len(fields) == 2 &&
		strings.EqualFold(fields[0], authorizationTypeBearer) {
		return fields[1]
	}
	if cfg.AllowQueryToken {
		return c.Query("token")
	}
	return ""
}

func abort(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
