package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

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
	id := requestIDFrom(c)

	for _, s := range sentinelStatus {
		if !errors.Is(err, s.err) {
			continue
		}
		msg := s.err.Error()
		// Invalid input carries the specific reason; others stay generic.
		if s.status == http.StatusBadRequest {
			msg = err.Error()
		}
		c.AbortWithStatusJSON(s.status, dto.ErrorResponse{Error: msg, RequestID: id})
		return
	}

	log.Error("unhandled error", "request_id", id, "path", c.Request.URL.Path, "error", err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Error: genericServerMessage, RequestID: id})
}
