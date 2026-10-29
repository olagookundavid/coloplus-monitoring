package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// requestAcceptance godoc
//
//	@Summary		Start a site's commissioning and acceptance
//	@Description	Only for a site still pending acceptance, and only once per site.
//	@Tags			acceptance
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Site id"
//	@Param			body	body		dto.RequestAcceptanceRequest	false	"Plan"
//	@Success		201		{object}	dto.AcceptanceDTO
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/acceptance [post]
func (a *api) requestAcceptance(c *gin.Context) {
	var req dto.RequestAcceptanceRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	acc, err := a.acceptance.Request(c.Request.Context(), actorFrom(c), services.RequestAcceptanceInput{
		SiteID: c.Param("id"), NextAction: req.NextAction, DueAt: req.DueAt,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAcceptanceDTO(acc, a.clock.Now()))
}

// getAcceptance godoc
//
//	@Summary	A site's acceptance with its history
//	@Tags		acceptance
//	@Produce	json
//	@Param		id	path		string	true	"Site id"
//	@Success	200	{object}	dto.AcceptanceDetailResponse
//	@Failure	404	{object}	dto.ErrorResponse
//	@Router		/api/v1/sites/{id}/acceptance [get]
func (a *api) getAcceptance(c *gin.Context) {
	v, err := a.acceptance.Get(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	resp := dto.AcceptanceDetailResponse{Acceptance: toAcceptanceDTO(v.Acceptance, a.clock.Now()), Events: make([]dto.AcceptanceEventDTO, 0, len(v.Events))}
	for _, e := range v.Events {
		resp.Events = append(resp.Events, toAcceptanceEventDTO(e))
	}
	c.JSON(http.StatusOK, resp)
}

// transitionAcceptance godoc
//
//	@Summary		Move a site's acceptance to its next stage
//	@Description	Coloplus: requested to built, then ready_for_inspection with the dossier; a rejected site goes back to built once fixed. Airtel: ready_for_inspection to inspected or rejected; inspected to accepted or rejected. Either side can put it on_hold with a reason and resume it by naming the stage it left. Only Airtel's accepted counts as acceptance, and it brings a pending site into service. Repeating a step changes nothing.
//	@Tags			acceptance
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Site id"
//	@Param			body	body		dto.TransitionAcceptanceRequest	true	"Next stage"
//	@Success		200		{object}	dto.AcceptanceDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/acceptance/transition [post]
func (a *api) transitionAcceptance(c *gin.Context) {
	var req dto.TransitionAcceptanceRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	acc, err := a.acceptance.Transition(c.Request.Context(), actorFrom(c), c.Param("id"), domain.AcceptanceStep{
		To: domain.AcceptanceStage(req.Stage), Note: req.Note, NextAction: req.NextAction, DueAt: req.DueAt,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toAcceptanceDTO(acc, a.clock.Now()))
}

// listAcceptances godoc
//
//	@Summary		The commissioning backlog
//	@Description	Sites still going through acceptance, the longest-waiting for inspection first.
//	@Tags			acceptance
//	@Produce		json
//	@Param			stage	query		string	false	"requested, built, ready_for_inspection, inspected, accepted, rejected or on_hold"
//	@Param			open	query		bool	false	"Leave out accepted sites"
//	@Param			overdue	query		bool	false	"Only those whose next action is overdue"
//	@Param			limit	query		int		false	"Page size (default 50, max 200)"
//	@Param			offset	query		int		false	"Offset"
//	@Success		200		{object}	dto.AcceptanceListResponse
//	@Router			/api/v1/acceptances [get]
func (a *api) listAcceptances(c *gin.Context) {
	in := services.ListAcceptancesInput{
		OpenOnly: c.Query("open") == "true", Overdue: c.Query("overdue") == "true",
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if st := c.Query("stage"); st != "" {
		in.Stages = []domain.AcceptanceStage{domain.AcceptanceStage(st)}
	}
	list, total, err := a.acceptance.List(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	now := a.clock.Now()
	resp := dto.AcceptanceListResponse{Acceptances: make([]dto.AcceptanceDTO, 0, len(list)), Total: total, Limit: limit, Offset: offset}
	for _, acc := range list {
		resp.Acceptances = append(resp.Acceptances, toAcceptanceDTO(acc, now))
	}
	c.JSON(http.StatusOK, resp)
}
