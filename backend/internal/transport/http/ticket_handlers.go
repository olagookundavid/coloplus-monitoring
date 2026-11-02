package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// raiseTicket godoc
//
//	@Summary		Raise a support ticket
//	@Description	Asks Coloplus for help with a site: a fault, a refill, an access problem. The priority is the one you ask for; Coloplus decides the severity of any incident it opens. Send client_key and retry safely: the same key returns the same ticket. Attachments are not supported yet.
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Site id"
//	@Param			body	body		dto.RaiseTicketRequest	true	"Ticket"
//	@Success		201		{object}	dto.TicketDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/tickets [post]
func (a *api) raiseTicket(c *gin.Context) {
	var req dto.RaiseTicketRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	t, err := a.tickets.Raise(c.Request.Context(), actor, services.TicketInput{
		SiteID: c.Param("id"), Category: domain.TicketCategory(req.Category), Priority: domain.TicketPriority(req.Priority),
		Subject: req.Subject, Description: req.Description, ContactEmail: req.ContactEmail, ClientKey: req.ClientKey,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTicketDTO(t, a.tickets.IsStaff(actor)))
}

// listTickets godoc
//
//	@Summary	List support tickets in your scope
//	@Tags		tickets
//	@Produce	json
//	@Param		site_id		query		string	false	"Only this site"
//	@Param		status		query		string	false	"open, in_progress or resolved"
//	@Param		category	query		string	false	"power_outage, low_fuel, generator_fault, equipment_fault, access_restriction or other"
//	@Param		priority	query		string	false	"low, medium, high or critical"
//	@Param		incident_id	query		string	false	"Only tickets linked to this incident"
//	@Param		open		query		bool	false	"Leave out resolved tickets"
//	@Param		limit		query		int		false	"Page size (default 50, max 200)"
//	@Param		offset		query		int		false	"Offset"
//	@Success	200			{object}	dto.TicketListResponse
//	@Router		/api/v1/tickets [get]
func (a *api) listTickets(c *gin.Context) {
	in := services.ListTicketsInput{
		SiteID: c.Query("site_id"), Category: domain.TicketCategory(c.Query("category")),
		Priority: domain.TicketPriority(c.Query("priority")), IncidentID: c.Query("incident_id"),
		OpenOnly: c.Query("open") == "true", Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	if st := c.Query("status"); st != "" {
		in.Statuses = []domain.TicketStatus{domain.TicketStatus(st)}
	}
	actor := actorFrom(c)
	list, total, err := a.tickets.List(c.Request.Context(), actor, in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	staff := a.tickets.IsStaff(actor)
	resp := dto.TicketListResponse{Tickets: make([]dto.TicketDTO, 0, len(list)), Total: total, Limit: limit, Offset: offset}
	for _, t := range list {
		resp.Tickets = append(resp.Tickets, toTicketDTO(t, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// ticketSummary godoc
//
//	@Summary	Headline ticket counts for your scope
//	@Tags		tickets
//	@Produce	json
//	@Success	200	{object}	dto.TicketSummaryResponse
//	@Router		/api/v1/tickets/summary [get]
func (a *api) ticketSummary(c *gin.Context) {
	n, err := a.tickets.Summary(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.TicketSummaryResponse{Open: n.Open, InProgress: n.InProgress, Resolved: n.Resolved, CriticalOpen: n.CriticalOpen})
}

// getTicket godoc
//
//	@Summary		One ticket with its history
//	@Description	Both sides' comments and every status change, oldest first. Coloplus's internal notes and staff details are left out for Airtel, and so is a linked incident until it is published to them.
//	@Tags			tickets
//	@Produce		json
//	@Param			id	path		string	true	"Ticket id"
//	@Success		200	{object}	dto.TicketDetailResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/tickets/{id} [get]
func (a *api) getTicket(c *gin.Context) {
	actor := actorFrom(c)
	v, err := a.tickets.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	staff := a.tickets.IsStaff(actor)
	resp := dto.TicketDetailResponse{Ticket: toTicketDTO(v.Ticket, staff), Events: make([]dto.TicketEventDTO, 0, len(v.Events))}
	for _, e := range v.Events {
		resp.Events = append(resp.Events, toTicketEventDTO(e, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// commentTicket godoc
//
//	@Summary		Comment on a ticket
//	@Description	Both sides can comment, also after it is resolved. Coloplus can mark a comment internal so Airtel never sees it.
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Ticket id"
//	@Param			body	body		dto.TicketCommentRequest	true	"Comment"
//	@Success		201		{object}	dto.TicketEventDTO
//	@Router			/api/v1/tickets/{id}/comments [post]
func (a *api) commentTicket(c *gin.Context) {
	var req dto.TicketCommentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	e, err := a.tickets.Comment(c.Request.Context(), actor, c.Param("id"), req.Body, req.Internal)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTicketEventDTO(e, a.tickets.IsStaff(actor)))
}

// transitionTicket godoc
//
//	@Summary		Move a ticket to its next status
//	@Description	Coloplus: open to in_progress or resolved, in_progress to resolved, and resolved back to in_progress. Resolving needs a note. Airtel: status open reopens a resolved ticket, with a note. Repeating a step that already happened changes nothing.
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Ticket id"
//	@Param			body	body		dto.TransitionTicketRequest		true	"Next status"
//	@Success		200		{object}	dto.TicketDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/tickets/{id}/transition [post]
func (a *api) transitionTicket(c *gin.Context) {
	var req dto.TransitionTicketRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	t, err := a.tickets.Transition(c.Request.Context(), actor, c.Param("id"), domain.TicketStep{
		To: domain.TicketStatus(req.Status), Note: req.Note,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toTicketDTO(t, a.tickets.IsStaff(actor)))
}

// linkTicketIncident godoc
//
//	@Summary		Link a ticket to the incident opened for it (Coloplus)
//	@Description	The incident must be at the same site. The link is noted internally; Airtel sees it on the ticket only once the incident has been published to them.
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Ticket id"
//	@Param			body	body		dto.LinkIncidentRequest		true	"Incident"
//	@Success		200		{object}	dto.TicketDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/tickets/{id}/incident [post]
func (a *api) linkTicketIncident(c *gin.Context) {
	var req dto.LinkIncidentRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	t, err := a.tickets.LinkIncident(c.Request.Context(), actor, c.Param("id"), req.IncidentID)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toTicketDTO(t, a.tickets.IsStaff(actor)))
}
