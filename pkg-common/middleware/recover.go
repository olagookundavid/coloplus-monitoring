package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recover turns a panic into a logged, generic 500 so internals never leak.
func Recover(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.Writer.Header().Set("Connection", "close")
				log.Error("panic recovered", "panic", rec, "method", c.Request.Method, "path", c.Request.URL.Path)
				abort(c, http.StatusInternalServerError, "the server hit a problem and could not complete your request")
			}
		}()
		c.Next()
	}
}
