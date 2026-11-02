package httpapi

import (
	"io"
	"net/http"
	"strings"

	apperrors "coloplus-backend/internal/errors"

	"github.com/gin-gonic/gin"
)

// resendWebhook receives the email provider's delivery events. It has no
// session; it is genuine only if the provider's signature checks out, and the
// body is not read as anything until it does. It is not in the API contract.
func (a *api) resendWebhook(c *gin.Context) {
	if !a.notifications.MailWebhookEnabled() {
		a.fail(c, apperrors.ErrNotFound)
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		a.fail(c, badRequest("the body could not be read"))
		return
	}
	headers := map[string]string{}
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}
	if err := a.notifications.HandleMailWebhook(c.Request.Context(), headers, body); err != nil {
		a.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
