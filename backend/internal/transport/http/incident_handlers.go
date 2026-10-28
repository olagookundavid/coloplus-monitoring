package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// incidentBody picks the view a person is entitled to: Coloplus staff see
// the internal record, Airtel sees only what has been published for them.
func (a *api) incidentBody(actor domain.Actor, i domain.Incident, updates []domain.IncidentUpdate) (*dto.IncidentDTO, *dto.PublishedIncidentDTO) {
	if actor.Can(domain.PermIncidentsRead) {
		d := toIncidentDTO(i, a.clock.Now())
		return &d, nil
	}
	p := toPublishedIncidentDTO(i, updates)
	return nil, &p
}

func (a *api) writeIncident(c *gin.Context, status int, i domain.Incident) {
	internal, published := a.incidentBody(actorFrom(c), i, nil)
	if internal != nil {
		c.JSON(status, internal)
		return
	}
	c.JSON(status, published)
}

// reportIncident godoc
//
//	@Summary		Report an incident at a site
//	@Description	Records a fault. Use impact no_outage for a fault that interrupted nothing, so it stays apart from service interruptions. The start time is when the fault began; the detection time defaults to it.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Site id"
//	@Param			body	body		dto.ReportIncidentRequest	true	"Incident"
//	@Success		201		{object}	dto.IncidentDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/incidents [post]
func (a *api) reportIncident(c *gin.Context) {
	var req dto.ReportIncidentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	i, err := a.incidents.Report(c.Request.Context(), actorFrom(c), services.IncidentInput{
		SiteID: c.Param("id"), Category: domain.IncidentCategory(req.Category), Severity: domain.Severity(req.Severity),
		Impact: domain.Impact(req.Impact), Title: req.Title, Description: req.Description,
		StartedAt: req.StartedAt, DetectedAt: req.DetectedAt,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeIncident(c, http.StatusCreated, i)
}

// listIncidents godoc
//
//	@Summary		List incidents in your scope
//	@Description	Airtel users see only incidents that have been published to them.
//	@Tags			incidents
//	@Produce		json
//	@Param			site_id		query		string	false	"Only this site"
//	@Param			owner_id	query		string	false	"Only incidents owned by this person"
//	@Param			status		query		string	false	"reported, acknowledged, assigned, in_progress, restored, verified or closed"
//	@Param			severity	query		string	false	"critical, major or minor"
//	@Param			open		query		bool	false	"Leave out closed incidents"
//	@Param			overdue		query		bool	false	"Only incidents whose promised update is overdue"
//	@Param			limit		query		int		false	"Page size (default 50, max 200)"
//	@Param			offset		query		int		false	"Offset"
//	@Success		200			{object}	dto.IncidentListResponse
//	@Router			/api/v1/incidents [get]
func (a *api) listIncidents(c *gin.Context) {
	in := services.ListIncidentsInput{
		SiteID: c.Query("site_id"), OwnerID: c.Query("owner_id"), Severity: domain.Severity(c.Query("severity")),
		OpenOnly: c.Query("open") == "true", Overdue: c.Query("overdue") == "true",
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if st := c.Query("status"); st != "" {
		in.Statuses = []domain.IncidentStatus{domain.IncidentStatus(st)}
	}
	actor := actorFrom(c)
	list, total, err := a.incidents.List(c.Request.Context(), actor, in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	resp := dto.IncidentListResponse{Total: total, Limit: limit, Offset: offset}
	for _, i := range list {
		internal, published := a.incidentBody(actor, i, nil)
		if internal != nil {
			resp.Incidents = append(resp.Incidents, *internal)
		} else {
			resp.Published = append(resp.Published, *published)
		}
	}
	c.JSON(http.StatusOK, resp)
}

// getIncident godoc
//
//	@Summary		One incident with its history
//	@Description	Staff get the full record and every update. Airtel gets the published view and the updates written for them.
//	@Tags			incidents
//	@Produce		json
//	@Param			id	path		string	true	"Incident id"
//	@Success		200	{object}	dto.IncidentDetailResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/incidents/{id} [get]
func (a *api) getIncident(c *gin.Context) {
	actor := actorFrom(c)
	v, err := a.incidents.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	internal, published := a.incidentBody(actor, v.Incident, v.Updates)
	resp := dto.IncidentDetailResponse{Incident: internal, Published: published}
	if internal != nil {
		for _, u := range v.Updates {
			resp.Updates = append(resp.Updates, toIncidentUpdateDTO(u))
		}
	}
	c.JSON(http.StatusOK, resp)
}

// assignIncident godoc
//
//	@Summary		Assign an owner
//	@Description	The owner must be an active person who can work on incidents at that site. Assigning a reported or acknowledged incident moves it to assigned. Optionally set the next action and when the next update is due.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Incident id"
//	@Param			body	body		dto.AssignIncidentRequest	true	"Owner and plan"
//	@Success		200		{object}	dto.IncidentDTO
//	@Router			/api/v1/incidents/{id}/assign [post]
func (a *api) assignIncident(c *gin.Context) {
	var req dto.AssignIncidentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	i, err := a.incidents.Assign(c.Request.Context(), actorFrom(c), c.Param("id"), services.AssignInput{
		OwnerID: req.OwnerID, NextAction: req.NextAction, NextUpdateDue: req.NextUpdateDue, Note: req.Note,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeIncident(c, http.StatusOK, i)
}

// transitionIncident godoc
//
//	@Summary		Move an incident to its next stage
//	@Description	Stages run reported, acknowledged, assigned, in_progress, restored, verified, closed. Restoration needs the time power was actually restored; a target does not count. Verification needs a new observation (what and when) from someone other than the person who recorded restoration. Closing needs a root cause. Repeating a step that already happened changes nothing.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Incident id"
//	@Param			body	body		dto.TransitionIncidentRequest	true	"Next stage"
//	@Success		200		{object}	dto.IncidentDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/incidents/{id}/transition [post]
func (a *api) transitionIncident(c *gin.Context) {
	var req dto.TransitionIncidentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	i, err := a.incidents.Transition(c.Request.Context(), actorFrom(c), c.Param("id"), domain.Transition{
		To: domain.IncidentStatus(req.Status), At: req.At, Note: req.Note, RootCause: req.RootCause,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeIncident(c, http.StatusOK, i)
}

// addIncidentUpdate godoc
//
//	@Summary		Add an update to an incident
//	@Description	Anyone who can work on the incident may add a note. Supervisors can also set the plan (next action, next update time, restoration target) and mark an update for Airtel, which needs the incident to be published first.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Incident id"
//	@Param			body	body		dto.IncidentUpdateRequest	true	"Update"
//	@Success		201		{object}	dto.IncidentUpdateDTO
//	@Router			/api/v1/incidents/{id}/updates [post]
func (a *api) addIncidentUpdate(c *gin.Context) {
	var req dto.IncidentUpdateRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	u, err := a.incidents.AddUpdate(c.Request.Context(), actorFrom(c), c.Param("id"), services.IncidentUpdateInput{
		Body: req.Body, CustomerVisible: req.CustomerVisible, NextAction: req.NextAction,
		NextUpdateDue: req.NextUpdateDue, RestorationTarget: req.RestorationTarget,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toIncidentUpdateDTO(u))
}

// publishIncident godoc
//
//	@Summary		Share an incident with Airtel
//	@Description	Does not wait for the daily report to be approved, so a critical incident can be notified at once. Safe to repeat.
//	@Tags			incidents
//	@Produce		json
//	@Param			id	path		string	true	"Incident id"
//	@Success		200	{object}	dto.IncidentDTO
//	@Router			/api/v1/incidents/{id}/publish [post]
func (a *api) publishIncident(c *gin.Context) {
	i, err := a.incidents.Publish(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeIncident(c, http.StatusOK, i)
}

// incidentSummary godoc
//
//	@Summary		Open, active, critical and overdue incident counts
//	@Tags			incidents
//	@Produce		json
//	@Success		200	{object}	dto.IncidentSummaryResponse
//	@Router			/api/v1/incidents/summary [get]
func (a *api) incidentSummary(c *gin.Context) {
	n, err := a.incidents.Summary(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.IncidentSummaryResponse{Open: n.Open, Active: n.Active, Critical: n.Critical, Overdue: n.Overdue})
}
