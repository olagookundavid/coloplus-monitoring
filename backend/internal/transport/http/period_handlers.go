package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func toPeriodDTO(c domain.PeriodClosure) dto.PeriodClosureDTO {
	return dto.PeriodClosureDTO{
		ID: c.ID, Month: c.Month.Format("2006-01"), ClosedBy: c.ClosedBy, ClosedAt: c.ClosedAt, RunID: c.RunID, Open: c.Open(),
		ReopenedBy: c.ReopenedBy, ReopenedAt: c.ReopenedAt, ReopenReason: c.ReopenReason,
	}
}

// listPeriods godoc
//
//	@Summary	Closed months
//	@Description	Every closure, newest first, reopened ones included (locked says which still hold).
//	@Tags		periods
//	@Produce	json
//	@Success	200	{object}	dto.PeriodListResponse
//	@Router		/api/v1/periods [get]
func (a *api) listPeriods(c *gin.Context) {
	l, err := a.periods.List(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	out := dto.PeriodListResponse{Periods: make([]dto.PeriodClosureDTO, len(l))}
	for i, p := range l {
		out.Periods[i] = toPeriodDTO(p)
	}
	a.respondNamed(c, http.StatusOK, out)
}

// closePeriod godoc
//
//	@Summary		Close a month
//	@Description	Locks a finished month once the grace period after its end has passed (PERIOD_CLOSE_GRACE_DAYS, default 5): no report for any of its days can then be added, corrected or approved until an administrator reopens it. Closing stores the month's calculation run first. Administrators only; recorded in the audit log.
//	@Tags			periods
//	@Produce		json
//	@Param			month	path		string	true	"Month, 2026-10"
//	@Success		201		{object}	dto.PeriodClosureDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/periods/{month}/close [post]
func (a *api) closePeriod(c *gin.Context) {
	month, err := services.ParseMonth(c.Param("month"))
	if err != nil {
		a.fail(c, err)
		return
	}
	closure, err := a.periods.Close(c.Request.Context(), actorFrom(c), month)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusCreated, toPeriodDTO(closure))
}

// reopenPeriod godoc
//
//	@Summary		Reopen a closed month
//	@Description	Lifts the lock so reports for the month can be corrected. The reason is required and kept in the audit log. Closing the month again stores a new run, and comparing it with the first shows what changed. Administrators only.
//	@Tags			periods
//	@Accept			json
//	@Produce		json
//	@Param			month	path		string						true	"Month, 2026-10"
//	@Param			body	body		dto.ReopenPeriodRequest		true	"Reason"
//	@Success		200		{object}	dto.PeriodClosureDTO
//	@Failure		404		{object}	dto.ErrorResponse
//	@Router			/api/v1/periods/{month}/reopen [post]
func (a *api) reopenPeriod(c *gin.Context) {
	month, err := services.ParseMonth(c.Param("month"))
	if err != nil {
		a.fail(c, err)
		return
	}
	var req dto.ReopenPeriodRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	closure, err := a.periods.Reopen(c.Request.Context(), actorFrom(c), month, req.Reason)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toPeriodDTO(closure))
}
