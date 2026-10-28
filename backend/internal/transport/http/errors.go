package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"coloplus-backend/internal/dto"
	apperrors "coloplus-backend/internal/errors"

	"github.com/gin-gonic/gin"
)

// sentinelStatus is the one place an application error becomes an HTTP status.
var sentinelStatus = []struct {
	err    error
	status int
}{
	{apperrors.ErrInvalidInput, http.StatusBadRequest},
	{apperrors.ErrInvalidCredentials, http.StatusUnauthorized},
	{apperrors.ErrInvalidCode, http.StatusUnauthorized},
	{apperrors.ErrAccountLocked, http.StatusTooManyRequests},
	{apperrors.ErrUnauthorized, http.StatusUnauthorized},
	{apperrors.ErrForbidden, http.StatusForbidden},
	{apperrors.ErrNotFound, http.StatusNotFound},
	{apperrors.ErrConflict, http.StatusConflict},
	{apperrors.ErrUnavailable, http.StatusServiceUnavailable},
}

const genericServerMessage = "the server hit a problem and could not complete your request"

// respondError writes err as JSON. Known errors keep their worded message;
// anything else is logged with the request id and returned as a generic 500
// so driver and internal detail never reaches the client.
func respondError(c *gin.Context, log *slog.Logger, err error) {
	if log == nil {
		log = slog.Default()
	}
	id := requestIDFrom(c)

	for _, s := range sentinelStatus {
		if !errors.Is(err, s.err) {
			continue
		}
		msg := reasonFor(err, s.err)
		c.AbortWithStatusJSON(s.status, dto.ErrorResponse{Error: msg, RequestID: id})
		return
	}

	log.Error("unhandled error", "request_id", id, "path", c.Request.URL.Path, "error", err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Error: genericServerMessage, RequestID: id})
}

// reasonFor returns the specific, worded reason a service attached with
// fmt.Errorf("%w: reason", sentinel), or the sentinel's own message. Errors
// that merely pass through a sentinel (a wrapped driver error) never expose
// their detail.
func reasonFor(err, sentinel error) string {
	prefix := sentinel.Error() + ": "
	if msg := err.Error(); strings.HasPrefix(msg, prefix) && len(msg) > len(prefix) {
		return msg[len(prefix):]
	}
	return sentinel.Error()
}
