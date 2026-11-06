package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// requestIncidentCorrection godoc
//
//	@Summary		Ask to correct an incident field
//	@Description	Corrects when a fault started or when it was actually restored. The current value is snapshotted server-side, never taken from the request. Refused once the incident is closed. Only one request per incident and field may be awaiting review at a time; that answers 409.
//	@Tags			corrections
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Incident id"
//	@Param			body	body		dto.RequestCorrectionRequest	true	"Correction"
//	@Success		201		{object}	dto.IncidentCorrectionDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/incidents/{id}/corrections [post]
func (a *api) requestIncidentCorrection(c *gin.Context) {
	var req dto.RequestCorrectionRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	corr, err := a.corrections.Request(c.Request.Context(), actorFrom(c), services.RequestCorrectionInput{
		IncidentID: c.Param("id"), Field: domain.CorrectionField(req.Field),
		CorrectedValue: req.CorrectedValue, Reason: req.Reason,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusCreated, toIncidentCorrectionDTO(corr))
}

// listIncidentCorrections godoc
//
//	@Summary	List incident corrections in your scope
//	@Tags		corrections
//	@Produce	json
//	@Param		site_id		query		string	false	"Only this site"
//	@Param		incident_id	query		string	false	"Only corrections for this incident"
//	@Param		status		query		string	false	"requested, approved or rejected"
//	@Param		limit		query		int		false	"Page size (default 50, max 200)"
//	@Param		offset		query		int		false	"Offset"
//	@Success	200			{object}	dto.IncidentCorrectionListResponse
//	@Router		/api/v1/incident-corrections [get]
func (a *api) listIncidentCorrections(c *gin.Context) {
	in := services.ListIncidentCorrectionsInput{
		SiteID: c.Query("site_id"), IncidentID: c.Query("incident_id"),
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if st := c.Query("status"); st != "" {
		in.Statuses = []domain.CorrectionStatus{domain.CorrectionStatus(st)}
	}
	list, total, err := a.corrections.List(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	resp := dto.IncidentCorrectionListResponse{
		Corrections: make([]dto.IncidentCorrectionDTO, 0, len(list)), Total: total, Limit: limit, Offset: offset,
	}
	for _, corr := range list {
		resp.Corrections = append(resp.Corrections, toIncidentCorrectionDTO(corr))
	}
	a.respondNamed(c, http.StatusOK, resp)
}

// getIncidentCorrection godoc
//
//	@Summary	One incident correction
//	@Tags		corrections
//	@Produce	json
//	@Param		id	path		string	true	"Correction id"
//	@Success	200	{object}	dto.IncidentCorrectionDTO
//	@Failure	404	{object}	dto.ErrorResponse
//	@Router		/api/v1/incident-corrections/{id} [get]
func (a *api) getIncidentCorrection(c *gin.Context) {
	corr, err := a.corrections.Get(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toIncidentCorrectionDTO(corr))
}

// approveIncidentCorrection godoc
//
//	@Summary		Approve an incident correction
//	@Description	Writes the corrected value to the incident and the decision to the correction, in one transaction. Never the person who requested it. Refused once the incident is closed.
//	@Tags			corrections
//	@Produce		json
//	@Param			id	path		string	true	"Correction id"
//	@Success		200	{object}	dto.IncidentCorrectionDTO
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/api/v1/incident-corrections/{id}/approve [post]
func (a *api) approveIncidentCorrection(c *gin.Context) {
	corr, err := a.corrections.Approve(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toIncidentCorrectionDTO(corr))
}

// rejectIncidentCorrection godoc
//
//	@Summary	Reject an incident correction
//	@Tags		corrections
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string							true	"Correction id"
//	@Param		body	body		dto.RejectCorrectionRequest	true	"Reason"
//	@Success	200		{object}	dto.IncidentCorrectionDTO
//	@Failure	400		{object}	dto.ErrorResponse
//	@Router		/api/v1/incident-corrections/{id}/reject [post]
func (a *api) rejectIncidentCorrection(c *gin.Context) {
	var req dto.RejectCorrectionRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	corr, err := a.corrections.Reject(c.Request.Context(), actorFrom(c), c.Param("id"), req.Note)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toIncidentCorrectionDTO(corr))
}
