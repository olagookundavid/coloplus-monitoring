package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// raiseDispute godoc
//
//	@Summary		Raise a dispute (Airtel reviewer)
//	@Description	Disputes a site, a period, a published report, or one interval of it. Only an approved report can be disputed. The same thing cannot be disputed twice while the first dispute is open; that answers 409.
//	@Tags			disputes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Site id"
//	@Param			body	body		dto.RaiseDisputeRequest		true	"Dispute"
//	@Success		201		{object}	dto.DisputeDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/disputes [post]
func (a *api) raiseDispute(c *gin.Context) {
	var req dto.RaiseDisputeRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	from, err := optionalDate(req.PeriodFrom)
	if err != nil {
		a.fail(c, err)
		return
	}
	to, err := optionalDate(req.PeriodTo)
	if err != nil {
		a.fail(c, err)
		return
	}
	d, err := a.disputes.Raise(c.Request.Context(), actorFrom(c), services.DisputeInput{
		SiteID: c.Param("id"), ReportID: req.ReportID, IntervalID: req.IntervalID,
		Metric: domain.DisputeMetric(req.Metric), PeriodFrom: from, PeriodTo: to,
		Category: domain.DisputeCategory(req.Category), Title: req.Title, Description: req.Description,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDisputeDTO(d, a.disputes.IsStaff(actorFrom(c))))
}

// listDisputes godoc
//
//	@Summary	List disputes in your scope
//	@Tags		disputes
//	@Produce	json
//	@Param		site_id		query		string	false	"Only this site"
//	@Param		report_id	query		string	false	"Only disputes about this report"
//	@Param		status		query		string	false	"open, under_review, awaiting_evidence, agreed_correction, resolved or closed"
//	@Param		open		query		bool	false	"Leave out closed disputes"
//	@Param		limit		query		int		false	"Page size (default 50, max 200)"
//	@Param		offset		query		int		false	"Offset"
//	@Success	200			{object}	dto.DisputeListResponse
//	@Router		/api/v1/disputes [get]
func (a *api) listDisputes(c *gin.Context) {
	in := services.ListDisputesInput{
		SiteID: c.Query("site_id"), ReportID: c.Query("report_id"), OpenOnly: c.Query("open") == "true",
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if st := c.Query("status"); st != "" {
		in.Statuses = []domain.DisputeStatus{domain.DisputeStatus(st)}
	}
	actor := actorFrom(c)
	list, total, err := a.disputes.List(c.Request.Context(), actor, in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	staff := a.disputes.IsStaff(actor)
	resp := dto.DisputeListResponse{Disputes: make([]dto.DisputeDTO, 0, len(list)), Total: total, Limit: limit, Offset: offset}
	for _, d := range list {
		resp.Disputes = append(resp.Disputes, toDisputeDTO(d, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// disputeSummary godoc
//
//	@Summary	Headline dispute counts for your scope
//	@Tags		disputes
//	@Produce	json
//	@Success	200	{object}	dto.DisputeSummaryResponse
//	@Router		/api/v1/disputes/summary [get]
func (a *api) disputeSummary(c *gin.Context) {
	n, err := a.disputes.Summary(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.DisputeSummaryResponse{Open: n.Open, WithColoplus: n.WithColoplus, WithAirtel: n.WithAirtel})
}

// getDispute godoc
//
//	@Summary		One dispute with its history
//	@Description	Both parties' comments and every status change, oldest first. Coloplus's internal notes and staff details are left out for Airtel.
//	@Tags			disputes
//	@Produce		json
//	@Param			id	path		string	true	"Dispute id"
//	@Success		200	{object}	dto.DisputeDetailResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/disputes/{id} [get]
func (a *api) getDispute(c *gin.Context) {
	actor := actorFrom(c)
	v, err := a.disputes.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	staff := a.disputes.IsStaff(actor)
	resp := dto.DisputeDetailResponse{Dispute: toDisputeDTO(v.Dispute, staff), Events: make([]dto.DisputeEventDTO, 0, len(v.Events))}
	for _, e := range v.Events {
		resp.Events = append(resp.Events, toDisputeEventDTO(e, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// commentDispute godoc
//
//	@Summary		Comment on a dispute
//	@Description	Both parties can comment. Coloplus can mark a comment internal so Airtel never sees it. An Airtel comment on a dispute waiting for their evidence hands it back to Coloplus.
//	@Tags			disputes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Dispute id"
//	@Param			body	body		dto.DisputeCommentRequest	true	"Comment"
//	@Success		201		{object}	dto.DisputeEventDTO
//	@Router			/api/v1/disputes/{id}/comments [post]
func (a *api) commentDispute(c *gin.Context) {
	var req dto.DisputeCommentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	e, err := a.disputes.Comment(c.Request.Context(), actor, c.Param("id"), req.Body, req.Internal)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toDisputeEventDTO(e, a.disputes.IsStaff(actor)))
}

// transitionDispute godoc
//
//	@Summary		Move a dispute to its next status
//	@Description	Coloplus: open to under_review, then awaiting_evidence, agreed_correction or resolved. Resolving needs an outcome; upheld needs the correction agreed first and, for a report, the corrected approved revision. Airtel: closed withdraws it, or acknowledges a resolution; under_review gives evidence or reopens a resolution. Repeating a step that already happened changes nothing. Coloplus's approval and Airtel's agreement are recorded separately.
//	@Tags			disputes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Dispute id"
//	@Param			body	body		dto.TransitionDisputeRequest	true	"Next status"
//	@Success		200		{object}	dto.DisputeDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/disputes/{id}/transition [post]
func (a *api) transitionDispute(c *gin.Context) {
	var req dto.TransitionDisputeRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	d, err := a.disputes.Transition(c.Request.Context(), actor, c.Param("id"), domain.DisputeStep{
		To: domain.DisputeStatus(req.Status), Note: req.Note, Outcome: domain.DisputeOutcome(req.Outcome),
		ResolutionReportID: req.ResolutionReportID,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toDisputeDTO(d, a.disputes.IsStaff(actor)))
}
