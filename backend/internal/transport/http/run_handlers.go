package httpapi

import (
	"net/http"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func toRunRowDTO(r domain.RunRow) dto.RunRowDTO {
	return dto.RunRowDTO{
		SiteID: r.SiteID, AirtelCode: r.AirtelCode, SiteName: r.SiteName,
		EligibleMinutes: r.EligibleMinutes, ServedMinutes: r.ServedMinutes, OutageMinutes: r.OutageMinutes,
		UnknownMinutes: r.UnknownMinutes, AssumedMinutes: r.AssumedMinutes, DaysInPeriod: r.DaysInPeriod, DaysReported: r.DaysReported,
		ReportedAvailability: r.ReportedAvailability, ConfirmedServedShare: r.ConfirmedServedShare, Coverage: r.Coverage,
		GeneratorHours: r.GeneratorHours, IncidentsStarted: r.IncidentsStarted, ReportsFingerprint: r.Fingerprint,
	}
}

func toRunDTO(r domain.CalculationRun, withRows bool) dto.CalculationRunDTO {
	out := dto.CalculationRunDTO{
		ID: r.ID, Kind: r.Kind, PeriodFrom: r.PeriodFrom.Format("2006-01-02"), PeriodTo: r.PeriodTo.Format("2006-01-02"),
		Region: r.Region, Label: r.Label, Formula: r.Formula, FormulaVersion: r.FormulaVersion,
		SiteCount: r.SiteCount, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
	if withRows {
		out.Rows = make([]dto.RunRowDTO, len(r.Rows))
		for i, x := range r.Rows {
			out.Rows[i] = toRunRowDTO(x)
		}
	}
	return out
}

// storeRun godoc
//
//	@Summary		Freeze a month's figures
//	@Description	Works out the month for every site (or one region) and stores it, with the formula version and a fingerprint of the approved reports behind each site's figures. A later correction does not change a stored run; it shows when the run is compared with now. Running it again makes a separate run. Coloplus staff who see every site and can export; recorded in the audit log.
//	@Tags			calculation-runs
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CalculationRunRequest	true	"Month"
//	@Success		201		{object}	dto.CalculationRunDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/calculation-runs [post]
func (a *api) storeRun(c *gin.Context) {
	var req dto.CalculationRunRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	month, err := time.Parse("2006-01", req.Month)
	if err != nil {
		a.fail(c, badRequest("the month must look like 2026-10"))
		return
	}
	run, err := a.exports.StoreRun(c.Request.Context(), actorFrom(c), month, req.Region, req.Label)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRunDTO(run, false))
}

// listRuns godoc
//
//	@Summary	Stored calculation runs, newest first
//	@Tags		calculation-runs
//	@Produce	json
//	@Param		limit	query		int	false	"Page size (default 50, max 200)"
//	@Param		offset	query		int	false	"Offset"
//	@Success	200		{object}	dto.CalculationRunListResponse
//	@Router		/api/v1/calculation-runs [get]
func (a *api) listRuns(c *gin.Context) {
	limit, offset := intQuery(c, "limit"), intQuery(c, "offset")
	runs, total, err := a.exports.Runs(c.Request.Context(), actorFrom(c), limit, offset)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset = services.Page(limit, offset)
	out := dto.CalculationRunListResponse{Runs: make([]dto.CalculationRunDTO, len(runs)), Total: total, Limit: limit, Offset: offset}
	for i, r := range runs {
		out.Runs[i] = toRunDTO(r, false)
	}
	c.JSON(http.StatusOK, out)
}

// getRun godoc
//
//	@Summary	One stored run with every site's figures
//	@Tags		calculation-runs
//	@Produce	json
//	@Param		id	path		string	true	"Run id"
//	@Success	200	{object}	dto.CalculationRunDTO
//	@Failure	404	{object}	dto.ErrorResponse
//	@Router		/api/v1/calculation-runs/{id} [get]
func (a *api) getRun(c *gin.Context) {
	run, err := a.exports.Run(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toRunDTO(run, true))
}

// compareRun godoc
//
//	@Summary		Set a stored run against today
//	@Description	Works the run's month out again now and lists only the sites whose figures differ, each with what the run said then and what is true now, and whether the approved reports behind it have changed (a correction or a later approval). If the formula version has changed since, formula_changed says so, because then a difference may come from the formula rather than from the reports. A run of a month still in progress cannot be compared once the month has moved on.
//	@Tags			calculation-runs
//	@Produce		json
//	@Param			id	path		string	true	"Run id"
//	@Success		200	{object}	dto.RunComparisonDTO
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/calculation-runs/{id}/compare [get]
func (a *api) compareRun(c *gin.Context) {
	cmp, err := a.exports.CompareRun(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	out := dto.RunComparisonDTO{
		Run: toRunDTO(cmp.Run, false), GeneratedAt: cmp.GeneratedAt, FormulaChanged: cmp.FormulaChanged,
		CurrentVersion: cmp.CurrentVersion, Unchanged: cmp.Unchanged, Changes: make([]dto.RunChangeDTO, len(cmp.Changes)),
	}
	for i, ch := range cmp.Changes {
		row := dto.RunChangeDTO{SiteID: ch.SiteID, AirtelCode: ch.AirtelCode, SiteName: ch.SiteName, ReportsChanged: ch.ReportsChanged}
		if ch.Then != nil {
			t := toRunRowDTO(*ch.Then)
			row.Then = &t
		}
		if ch.Now != nil {
			n := toRunRowDTO(*ch.Now)
			row.Now = &n
		}
		out.Changes[i] = row
	}
	c.JSON(http.StatusOK, out)
}
