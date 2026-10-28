package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

// requestID tags every request so a log line can be tied to a response.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		id := hex.EncodeToString(buf)
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestIDFrom(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

// accessLog stays quiet for successful requests and logs 4xx and 5xx only.
func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		if status < 400 {
			return
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		log.Log(c.Request.Context(), level, "request failed",
			"request_id", requestIDFrom(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"took", time.Since(start),
		)
	}
}

func securityHeaders(production bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		if production {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// corsMiddleware allows exactly the configured origins. With none configured
// no CORS headers are sent at all, which is what a same-origin setup wants.
func corsMiddleware(origins []string) gin.HandlerFunc {
	if len(origins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
