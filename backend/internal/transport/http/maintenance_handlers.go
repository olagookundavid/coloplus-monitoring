package httpapi

import (
	"net/http"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func (a *api) toMaintenanceDTO(actor domain.Actor, w domain.MaintenanceWindow) dto.MaintenanceDTO {
	out := dto.MaintenanceDTO{
		ID: w.ID, SiteID: w.SiteID, StartsAt: w.StartsAt.In(a.loc), EndsAt: w.EndsAt.In(a.loc),
		Minutes: int(w.Duration() / time.Minute), Reason: w.Reason, Status: string(w.Status),
		RequestedAt: w.RequestedAt, DecidedAt: w.DecidedAt, DecisionNote: w.DecisionNote,
	}
	// Who asked and who decided is Coloplus's own information.
	if !actor.Can(domain.PermPublishedRead) {
		out.RequestedBy, out.DecidedBy = w.RequestedBy, w.DecidedBy
	}
	return out
}

// requestMaintenance godoc
//
//	@Summary		Ask for planned maintenance
//	@Description	Asks for a window of planned work at a site. Once another person approves it, the time is taken out of the site's eligible time, so it is held to rules: asked for at least notice_hours ahead, at most max_hours long, within one day, within monthly_cap_hours a month for the site, and not overlapping another window. The rules are returned by the list. Another person has to approve it, never the one who asked.
//	@Tags			maintenance
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Site id"
//	@Param			body	body		dto.MaintenanceRequest		true	"Window"
//	@Success		201		{object}	dto.MaintenanceDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/maintenance [post]
func (a *api) requestMaintenance(c *gin.Context) {
	var req dto.MaintenanceRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	w, err := a.maintenance.Request(c.Request.Context(), actor, c.Param("id"), services.MaintenanceInput{StartsAt: req.StartsAt, EndsAt: req.EndsAt, Reason: req.Reason})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, a.toMaintenanceDTO(actor, w))
}

// listMaintenance godoc
//
//	@Summary	Planned maintenance windows
//	@Tags		maintenance
//	@Produce	json
//	@Param		site_id	query		string	false	"One site"
//	@Param		status	query		string	false	"requested, approved, rejected or cancelled"
//	@Param		from	query		string	false	"Windows starting on or after this day, 2026-12-01"
//	@Param		to		query		string	false	"Windows starting before the end of this day"
//	@Param		limit	query		int		false	"Page size (default 50, max 200)"
//	@Param		offset	query		int		false	"Offset"
//	@Success	200		{object}	dto.MaintenanceListResponse
//	@Router		/api/v1/maintenance [get]
func (a *api) listMaintenance(c *gin.Context) {
	in := services.ListMaintenanceInput{
		SiteID: c.Query("site_id"), Status: domain.MaintenanceStatus(c.Query("status")),
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if v := c.Query("from"); v != "" {
		d, err := services.ParseDate(v)
		if err != nil {
			a.fail(c, err)
			return
		}
		start, _ := domain.DayBounds(d, a.loc)
		in.From = &start
	}
	if v := c.Query("to"); v != "" {
		d, err := services.ParseDate(v)
		if err != nil {
			a.fail(c, err)
			return
		}
		_, end := domain.DayBounds(d, a.loc)
		in.To = &end
	}
	actor := actorFrom(c)
	list, total, err := a.maintenance.List(c.Request.Context(), actor, in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	r := a.maintenance.Rules()
	out := dto.MaintenanceListResponse{
		Windows: make([]dto.MaintenanceDTO, len(list)), Total: total, Limit: limit, Offset: offset,
		Rules: dto.MaintenanceRulesDTO{NoticeHours: r.NoticeHours, MaxHours: r.MaxHours, MonthlyCapHours: r.MonthlyCapHours},
	}
	for i, w := range list {
		out.Windows[i] = a.toMaintenanceDTO(actor, w)
	}
	c.JSON(http.StatusOK, out)
}

func (a *api) decideMaintenance(c *gin.Context, do func(*services.MaintenanceService, *gin.Context, domain.Actor, string, string) (domain.MaintenanceWindow, error)) {
	var req dto.MaintenanceDecisionRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	w, err := do(a.maintenance, c, actor, c.Param("id"), req.Note)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, a.toMaintenanceDTO(actor, w))
}

// approveMaintenance godoc
//
//	@Summary		Approve a planned maintenance window
//	@Description	Never the person who asked. From approval the time is taken out of the site's eligible time.
//	@Tags			maintenance
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Window id"
//	@Param			body	body		dto.MaintenanceDecisionRequest	false	"Optional note"
//	@Success		200		{object}	dto.MaintenanceDTO
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/maintenance/{id}/approve [post]
func (a *api) approveMaintenance(c *gin.Context) {
	a.decideMaintenance(c, func(s *services.MaintenanceService, c *gin.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
		return s.Approve(c.Request.Context(), actor, id, note)
	})
}

// rejectMaintenance godoc
//
//	@Summary	Reject a planned maintenance window
//	@Tags		maintenance
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string							true	"Window id"
//	@Param		body	body		dto.MaintenanceDecisionRequest	true	"Why"
//	@Success	200		{object}	dto.MaintenanceDTO
//	@Router		/api/v1/maintenance/{id}/reject [post]
func (a *api) rejectMaintenance(c *gin.Context) {
	a.decideMaintenance(c, func(s *services.MaintenanceService, c *gin.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
		return s.Reject(c.Request.Context(), actor, id, note)
	})
}

// cancelMaintenance godoc
//
//	@Summary		Cancel a planned maintenance window
//	@Description	Before it starts. The person who asked, or an approver, may cancel; the reason is required. An approved window that is cancelled gives its time back to the site's eligible time.
//	@Tags			maintenance
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Window id"
//	@Param			body	body		dto.MaintenanceDecisionRequest	true	"Why"
//	@Success		200		{object}	dto.MaintenanceDTO
//	@Router			/api/v1/maintenance/{id}/cancel [post]
func (a *api) cancelMaintenance(c *gin.Context) {
	a.decideMaintenance(c, func(s *services.MaintenanceService, c *gin.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
		return s.Cancel(c.Request.Context(), actor, id, note)
	})
}
