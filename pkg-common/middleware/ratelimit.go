package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig sizes the per-client limiter.
type RateLimitConfig struct {
	Enabled bool
	Rps     float64
	Burst   int
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit limits each client IP. The idle-client sweeper stops when ctx ends.
//
// The client IP prefers Cloudflare's CF-Connecting-IP header, which is only
// trustworthy because the origin accepts traffic from Cloudflare ranges alone.
func RateLimit(ctx context.Context, cfg RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*clientLimiter)
	)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				for ip, cl := range clients {
					if time.Since(cl.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return func(c *gin.Context) {
		ip := clientIP(c)

		mu.Lock()
		cl, ok := clients[ip]
		if !ok {
			cl = &clientLimiter{limiter: rate.NewLimiter(rate.Limit(cfg.Rps), cfg.Burst)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		allowed := cl.limiter.Allow()
		mu.Unlock()

		if !allowed {
			abort(c, http.StatusTooManyRequests, "too many requests, please slow down and try again shortly")
			return
		}
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}
