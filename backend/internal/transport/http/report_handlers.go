package httpapi

import (
	"net/http"
	"strings"
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
		services.ReportInput{
			Notes: req.Notes, Intervals: fromIntervalDTOs(req.Intervals),
			PowerStatus: parsePowerStatus(req.PowerStatus), GeneratorHours: req.GeneratorHours,
		})
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
		services.ReportInput{
			Notes: req.Notes, Intervals: fromIntervalDTOs(req.Intervals),
			PowerStatus: parsePowerStatus(req.PowerStatus), GeneratorHours: req.GeneratorHours,
		})
	if err != nil {
		a.fail(c, err)
		return
	}
	a.writeReport(c, http.StatusOK, r)
}

// submitReport godoc
//
//	@Summary		Submit a draft for approval
//	@Description	Safe to repeat: submitting an already submitted report changes nothing. A report submitted after its site would already show "Awaiting update" (the day ended more than the site's freshness window ago) needs late_reason; without it the answer is 400 saying so. A correction does not.
//	@Tags			daily-reports
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Report id"
//	@Param			body	body		dto.SubmitRequest	false	"Why it is late, when it is"
//	@Success		200		{object}	dto.ReportDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/daily-reports/{id}/submit [post]
func (a *api) submitReport(c *gin.Context) {
	var req dto.SubmitRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	r, err := a.reports.Submit(c.Request.Context(), actorFrom(c), c.Param("id"), req.LateReason)
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
	actor := actorFrom(c)
	internal, published := a.reportBody(actor, view.Report)
	open, err := a.disputes.OpenByReport(c.Request.Context(), actor, []string{view.Report.ID})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ReportDetailResponse{
		Report: internal, Published: published,
		Summary: toSummaryDTO(view.Summary), Calculation: toCalculationDTO(view.Calculation, a.loc),
		OpenDisputes: open[view.Report.ID],
	})
}

// listReports godoc
//
//	@Summary	List daily reports in your scope
//	@Tags		daily-reports
//	@Produce	json
//	@Param		site_id	query		string	false	"Only this site"
//	@Param		mine	query		bool	false	"Only reports you authored"
//	@Param		from	query		string	false	"First report date, 2026-12-01"
//	@Param		to		query		string	false	"Last report date"
//	@Param		status	query		string	false	"draft, submitted, approved, rejected or superseded"
//	@Param		limit	query		int		false	"Page size (default 50, max 200)"
//	@Param		offset	query		int		false	"Offset"
//	@Success	200		{object}	dto.ReportListResponse
//	@Router		/api/v1/daily-reports [get]
func (a *api) listReports(c *gin.Context) {
	in := services.ListReportsInput{SiteID: c.Query("site_id"), Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset")}
	if c.Query("mine") == "true" {
		in.AuthorID = actorFrom(c).UserID
	}
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
	ids := make([]string, len(reports))
	for i, r := range reports {
		ids[i] = r.ID
	}
	if resp.OpenDisputes, err = a.disputes.OpenByReport(c.Request.Context(), actorFrom(c), ids); err != nil {
		a.fail(c, err)
		return
	}
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

// parsePowerStatus reads the reporter's simplified toggle. An empty or
// unrecognised value is left to domain.ValidateIntervals to reject, worded
// for the reporter, rather than failing early here with a generic error.
func parsePowerStatus(v string) *domain.IntervalState {
	if v == "" {
		return nil
	}
	st := domain.IntervalState(v)
	return &st
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
//	@Summary		Portfolio overview for a day or a period
//	@Description	Availability pools active sites by eligible minutes over the period. A day with no approved report counts its window as unknown; it never appears as available. Give `date` for one day, or `from` and `to` (at most 92 days). `region` narrows the sites; `regions` lists what is on offer. Each site row says how many of the days were reported and, when its report is current, the source it was last observed running on.
//	@Tags			dashboard
//	@Produce		json
//	@Param			date	query		string	false	"One reporting day, defaults to today (Africa/Lagos)"
//	@Param			from	query		string	false	"First day of a period; give with to"
//	@Param			to		query		string	false	"Last day of a period (not in the future)"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{object}	dto.OverviewResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/dashboard/overview [get]
func (a *api) overview(c *gin.Context) {
	p, err := a.dashboardPeriod(c, true)
	if err != nil {
		a.fail(c, err)
		return
	}
	o, err := a.dashboard.Overview(c.Request.Context(), actorFrom(c), p)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toOverviewResponse(o, a.loc))
}

// mySites godoc
//
//	@Summary		The reporter's own site grid
//	@Description	Today's status, a trailing 7-day health strip, pooled 30-day availability and open-incident count, one card per assigned site. A site with no approved report in the window has a null availability -- never 0 -- and every bar red.
//	@Tags			dashboard
//	@Produce		json
//	@Success		200	{object}	dto.MySitesResponse
//	@Router			/api/v1/dashboard/my-sites [get]
func (a *api) mySites(c *gin.Context) {
	m, err := a.dashboard.MySites(c.Request.Context(), actorFrom(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toMySitesResponse(m))
}

// trend godoc
//
//	@Summary		Portfolio availability day by day
//	@Description	One point per reporting day for the active sites in scope, pooled by eligible minutes. A day on which nothing was reported has null availability: draw a gap, not a line at zero or 100. Daily is the finest grain the reports give.
//	@Tags			dashboard
//	@Produce		json
//	@Param			from	query		string	true	"First day"
//	@Param			to		query		string	true	"Last day (not in the future); at most 92 days"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{object}	dto.TrendResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/dashboard/trend [get]
func (a *api) trend(c *gin.Context) {
	p, err := a.dashboardPeriod(c, false)
	if err != nil {
		a.fail(c, err)
		return
	}
	t, err := a.dashboard.Trend(c.Request.Context(), actorFrom(c), p)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toTrendResponse(t))
}

// dashboardPeriod reads date, or from and to, and the region. With
// allowDefault the period is today when none is given.
func (a *api) dashboardPeriod(c *gin.Context, allowDefault bool) (services.Period, error) {
	p := services.Period{Region: strings.TrimSpace(c.Query("region"))}
	date, from, to := c.Query("date"), c.Query("from"), c.Query("to")
	var err error
	switch {
	case date != "" && (from != "" || to != ""):
		return p, badRequest("give either date, or from and to")
	case date != "":
		if p.From, err = services.ParseDate(date); err != nil {
			return p, err
		}
		p.To = p.From
	case from != "" || to != "":
		if from == "" || to == "" {
			return p, badRequest("give both from and to")
		}
		if p.From, err = services.ParseDate(from); err != nil {
			return p, err
		}
		if p.To, err = services.ParseDate(to); err != nil {
			return p, err
		}
	case allowDefault:
		y, m, d := a.clock.Now().In(a.loc).Date()
		p.From = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		p.To = p.From
	default:
		return p, badRequest("give the first and last day")
	}
	return p, nil
}
