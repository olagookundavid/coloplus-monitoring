package httpapi

import (
	"net/http"

	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// listAlerts godoc
//
//	@Summary		Your alerts
//	@Description	The alerts waiting for you, newest first: late reports, overdue incidents, low fuel, replies on tickets and disputes. Each stays until you acknowledge it. You only ever see your own. unread is the number you have not acknowledged, for a badge.
//	@Tags			alerts
//	@Produce		json
//	@Param			unread	query		bool	false	"Only those not yet acknowledged"
//	@Param			limit	query		int		false	"Page size (default 50, max 200)"
//	@Param			offset	query		int		false	"Offset"
//	@Success		200		{object}	dto.AlertListResponse
//	@Router			/api/v1/alerts [get]
func (a *api) listAlerts(c *gin.Context) {
	limit, offset := intQuery(c, "limit"), intQuery(c, "offset")
	l, err := a.alerts.List(c.Request.Context(), actorFrom(c), c.Query("unread") == "true", limit, offset)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset = services.Page(limit, offset)
	resp := dto.AlertListResponse{Alerts: make([]dto.AlertDTO, 0, len(l.Alerts)), Total: l.Total, Unread: l.Unread, Limit: limit, Offset: offset}
	for _, al := range l.Alerts {
		resp.Alerts = append(resp.Alerts, dto.AlertDTO{
			ID: al.ID, Kind: al.Kind, Title: al.Title, Body: al.Body, CreatedAt: al.CreatedAt, AcknowledgedAt: al.AcknowledgedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// acknowledgeAlert godoc
//
//	@Summary		Acknowledge one of your alerts
//	@Description	Safe to repeat. Someone else's alert, or one that does not exist, is not found.
//	@Tags			alerts
//	@Param			id	path	string	true	"Alert id"
//	@Success		204
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/alerts/{id}/acknowledge [post]
func (a *api) acknowledgeAlert(c *gin.Context) {
	if err := a.alerts.Acknowledge(c.Request.Context(), actorFrom(c), c.Param("id")); err != nil {
		a.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// acknowledgeAllAlerts godoc
//
//	@Summary	Acknowledge all your open alerts
//	@Tags		alerts
//	@Produce	json
//	@Success	200	{object}	dto.AcknowledgeAllResponse
//	@Router		/api/v1/alerts/acknowledge-all [post]
func (a *api) acknowledgeAllAlerts(c *gin.Context) {
	n, err := a.alerts.AcknowledgeAll(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.AcknowledgeAllResponse{Acknowledged: n})
}
