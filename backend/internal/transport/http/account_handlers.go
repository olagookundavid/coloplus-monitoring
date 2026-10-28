package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// forgotPassword godoc
//
//	@Summary		Ask for a password reset link
//	@Description	Always answers the same way, whether or not the address has an account, so it cannot be used to find out who does. The link works once and expires in an hour.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.ForgotPasswordRequest	true	"Email"
//	@Success		202		{object}	dto.MessageResponse
//	@Router			/api/v1/auth/forgot-password [post]
func (a *api) forgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	if err := a.accounts.RequestReset(c.Request.Context(), req.Email); err != nil {
		// Logged by the mapper; the caller still gets the neutral answer so a
		// failure cannot reveal whether the account exists.
		a.log.Error("requesting password reset", "error", err)
	}
	c.JSON(http.StatusAccepted, dto.MessageResponse{Message: "if that address has an account, a reset link is on its way"})
}

// setPassword godoc
//
//	@Summary		Choose a password with an invitation or reset link
//	@Description	The token comes from the emailed link and works once. Choosing a password signs out every earlier session.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SetPasswordRequest	true	"Token and new password"
//	@Success		200		{object}	dto.MessageResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/set-password [post]
func (a *api) setPassword(c *gin.Context) {
	var req dto.SetPasswordRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	if err := a.accounts.SetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "your password is set; you can sign in now"})
}

// resendInvite godoc
//
//	@Summary		Send a fresh invitation (administrators)
//	@Description	For someone who has not chosen a password yet. The earlier link stops working.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string	true	"User id"
//	@Success		202	{object}	dto.MessageResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/api/v1/users/{id}/invite [post]
func (a *api) resendInvite(c *gin.Context) {
	if err := a.users.ResendInvite(c.Request.Context(), actorFrom(c), c.Param("id")); err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, dto.MessageResponse{Message: "invitation queued"})
}

// listNotifications godoc
//
//	@Summary		Email delivery status (administrators)
//	@Description	Failed emails stay here until someone retries them. The message text is never returned.
//	@Tags			notifications
//	@Produce		json
//	@Param			status	query		string	false	"pending, sent or failed"
//	@Param			limit	query		int		false	"Page size (default 50, max 200)"
//	@Param			offset	query		int		false	"Offset"
//	@Success		200		{object}	dto.NotificationListResponse
//	@Router			/api/v1/notifications [get]
func (a *api) listNotifications(c *gin.Context) {
	limit, offset := intQuery(c, "limit"), intQuery(c, "offset")
	items, total, err := a.notifications.List(c.Request.Context(), actorFrom(c),
		domain.NotificationStatus(c.Query("status")), limit, offset)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset = services.Page(limit, offset)
	resp := dto.NotificationListResponse{Notifications: []dto.NotificationDTO{}, Total: total, Limit: limit, Offset: offset}
	for _, n := range items {
		resp.Notifications = append(resp.Notifications, dto.NotificationDTO{
			ID: n.ID, Kind: n.Kind, Recipient: n.Recipient, Subject: n.Subject, Status: string(n.Status),
			Attempts: n.Attempts, LastError: n.LastError, CreatedAt: n.CreatedAt, SentAt: n.SentAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// retryNotification godoc
//
//	@Summary		Retry a failed email (administrators)
//	@Tags			notifications
//	@Produce		json
//	@Param			id	path		string	true	"Notification id"
//	@Success		202	{object}	dto.MessageResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/api/v1/notifications/{id}/retry [post]
func (a *api) retryNotification(c *gin.Context) {
	if err := a.notifications.Retry(c.Request.Context(), actorFrom(c), c.Param("id")); err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, dto.MessageResponse{Message: "queued to send again"})
}
