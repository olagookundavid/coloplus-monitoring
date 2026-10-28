package httpapi

import (
	"net/http"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// reportBody picks the view a person is entitled to: Coloplus staff see the
// internal report, Airtel sees only the published facts.
func (a *api) reportBody(actor domain.Actor, r domain.DailyReport) (*dto.ReportDTO, *dto.PublishedReportDTO) {
	if actor.Can(domain.PermReportsRead) {
		d := toReportDTO(r, a.loc)
		return &d, nil
	}
	p := toPublishedDTO(r, a.loc)
	return nil, &p
}

func (a *api) writeReport(c *gin.Context, status int, r domain.DailyReport) {
	internal, published := a.reportBody(actorFrom(c), r)
	if internal != nil {
		c.JSON(status, internal)
		return
	}
	c.JSON(status, published)
}

// createReport godoc
//
//	@Summary		Start a daily report
//	@Description	Creates a draft for a site and reporting day. Intervals must stay within the day (Africa/Lagos), be whole minutes, and not end in the future. A served interval names its source; an outage has none. Served and outage intervals may not overlap.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Site id"
//	@Param			body	body		dto.CreateReportRequest	true	"Report"
//	@Success		201		{object}	dto.ReportDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/daily-reports [post]
func (a *api) createReport(c *gin.Context) {
	var req dto.CreateReportRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	date, err := services.ParseDate(req.ReportDate)
	if err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.CreateDraft(c.Request.Context(), actorFrom(c), c.Param("id"), date,
		services.ReportInput{Notes: req.Notes, Intervals: fromIntervalDTOs(req.Intervals)})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusCreated, r)
}

// saveReport godoc
//
//	@Summary		Save a draft
//	@Description	Replaces the notes and intervals. Send the row_version you loaded; if someone else changed the report meanwhile you get a conflict. Saving a returned report makes it a draft again.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Report id"
//	@Param			body	body		dto.SaveReportRequest	true	"Report"
//	@Success		200		{object}	dto.ReportDTO
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/daily-reports/{id} [put]
func (a *api) saveReport(c *gin.Context) {
	var req dto.SaveReportRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.SaveDraft(c.Request.Context(), actorFrom(c), c.Param("id"), req.RowVersion,
		services.ReportInput{Notes: req.Notes, Intervals: fromIntervalDTOs(req.Intervals)})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusOK, r)
}

// submitReport godoc
//
//	@Summary		Submit a draft for approval
//	@Description	Safe to repeat: submitting an already submitted report changes nothing.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Report id"
//	@Success		200	{object}	dto.ReportDTO
//	@Router			/api/v1/daily-reports/{id}/submit [post]
func (a *api) submitReport(c *gin.Context) {
	r, err := a.reports.Submit(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusOK, r)
}

// approveReport godoc
//
//	@Summary		Approve and publish a submitted report
//	@Description	The approver cannot be the author. Approving a correction supersedes the report it corrects.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Report id"
//	@Param			body	body		dto.ReviewRequest	false	"Optional comment"
//	@Success		200		{object}	dto.ReportDTO
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Router			/api/v1/daily-reports/{id}/approve [post]
func (a *api) approveReport(c *gin.Context) {
	var req dto.ReviewRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.Approve(c.Request.Context(), actorFrom(c), c.Param("id"), req.Comment)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusOK, r)
}

// rejectReport godoc
//
//	@Summary	Return a submitted report to its author
//	@Tags		daily-reports
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Report id"
//	@Param		body	body		dto.RejectRequest	true	"Reason"
//	@Success	200		{object}	dto.ReportDTO
//	@Router		/api/v1/daily-reports/{id}/reject [post]
func (a *api) rejectReport(c *gin.Context) {
	var req dto.RejectRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.Reject(c.Request.Context(), actorFrom(c), c.Param("id"), req.Reason)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusOK, r)
}

// startCorrection godoc
//
//	@Summary		Start a correction of an approved report
//	@Description	Creates a new draft revision linked to the approved one. The approved report is untouched until the correction is approved by someone other than its author.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Approved report id"
//	@Param			body	body		dto.CorrectionRequest	true	"Reason"
//	@Success		201		{object}	dto.ReportDTO
//	@Router			/api/v1/daily-reports/{id}/corrections [post]
func (a *api) startCorrection(c *gin.Context) {
	var req dto.CorrectionRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.StartCorrection(c.Request.Context(), actorFrom(c), c.Param("id"), req.Reason)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusCreated, r)
}

// getReport godoc
//
//	@Summary		One report with its calculated summary
//	@Description	Airtel users receive the published view of approved reports only.
//	@Tags			daily-reports
//	@Produce		json
//	@Param			id	path		string	true	"Report id"
//	@Success		200	{object}	dto.ReportDetailResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/daily-reports/{id} [get]
func (a *api) getReport(c *gin.Context) {
	view, err := a.reports.Get(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	internal, published := a.reportBody(actorFrom(c), view.Report)
	c.JSON(http.StatusOK, dto.ReportDetailResponse{
		Report: internal, Published: published,
		Summary: toSummaryDTO(view.Summary), Calculation: toCalculationDTO(view.Calculation, a.loc),
	})
}

// listReports godoc
//
//	@Summary	List daily reports in your scope
//	@Tags		daily-reports
//	@Produce	json
//	@Param		site_id	query		string	false	"Only this site"
//	@Param		from	query		string	false	"First report date, 2026-12-01"
//	@Param		to		query		string	false	"Last report date"
//	@Param		status	query		string	false	"draft, submitted, approved, rejected or superseded"
//	@Param		limit	query		int		false	"Page size (default 50, max 200)"
//	@Param		offset	query		int		false	"Offset"
//	@Success	200		{object}	dto.ReportListResponse
//	@Router		/api/v1/daily-reports [get]
func (a *api) listReports(c *gin.Context) {
	in := services.ListReportsInput{SiteID: c.Query("site_id"), Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset")}
	var err error
	if in.From, err = optionalDate(c.Query("from")); err != nil {
		a.fail(c, err)
		return
	}
	if in.To, err = optionalDate(c.Query("to")); err != nil {
		a.fail(c, err)
		return
	}
	if st := c.Query("status"); st != "" {
		in.Statuses = []domain.ReportStatus{domain.ReportStatus(st)}
	}
	reports, total, err := a.reports.List(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	resp := dto.ReportListResponse{Total: total, Limit: limit, Offset: offset}
	for _, r := range reports {
		internal, published := a.reportBody(actorFrom(c), r)
		if internal != nil {
			resp.Reports = append(resp.Reports, *internal)
		} else {
			resp.Published = append(resp.Published, *published)
		}
	}
	c.JSON(http.StatusOK, resp)
}

func optionalDate(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := services.ParseDate(v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// overview godoc
//
//	@Summary		Portfolio overview for a day
//	@Description	Availability pools active sites by eligible minutes. A site with no approved report for the day counts its window as unknown; it never appears as available.
//	@Tags			dashboard
//	@Produce		json
//	@Param			date	query		string	false	"Reporting date, defaults to today (Africa/Lagos)"
//	@Success		200		{object}	dto.OverviewResponse
//	@Router			/api/v1/dashboard/overview [get]
func (a *api) overview(c *gin.Context) {
	date, err := a.dashboardDate(c.Query("date"))
	if err != nil {
		a.fail(c, err)
		return
	}
	o, err := a.dashboard.Overview(c.Request.Context(), actorFrom(c), date)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toOverviewResponse(o, a.loc))
}

func (a *api) dashboardDate(v string) (time.Time, error) {
	if v != "" {
		return services.ParseDate(v)
	}
	y, m, d := a.clock.Now().In(a.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
}
