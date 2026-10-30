package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	apperrors "coloplus-backend/internal/errors"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 1 << 20

func badRequest(msg string) error {
	return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, msg)
}

// bindJSON reads a required JSON body.
func bindJSON(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return badRequest("the request body could not be read; check the fields and try again")
	}
	return nil
}

// bindOptionalJSON reads a JSON body that may be empty.
func bindOptionalJSON(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil && !errors.Is(err, io.EOF) {
		return badRequest("the request body could not be read; check the fields and try again")
	}
	return nil
}

// jsonWrites refuses state-changing requests that are not JSON. A browser
// cannot send application/json cross-site without a CORS preflight, so this
// doubles as a CSRF guard for cookie-authenticated calls.
func jsonWrites() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
				abortWith(c, badRequest("send requests with the header Content-Type: application/json"))
				return
			}
		}
		c.Next()
	}
}

// limitBody caps request bodies so a client cannot stream unbounded data.
func limitBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}
