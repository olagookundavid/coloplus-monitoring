package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// writeCSV sends a table as a download with the byte order mark Excel needs.
func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	_, _ = c.Writer.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(c.Writer)
	_ = w.Write(header)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
}

func itoa(n int) string { return strconv.Itoa(n) }

func hours1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

func utcTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

var monthlyHeader = []string{
	"site_id", "airtel_code", "coloplus_code", "site_name", "region", "responsible_team", "site_status",
	"period_from", "period_to", "time_zone",
	"eligible_minutes", "served_minutes", "outage_minutes", "unknown_minutes", "assumed_available_minutes",
	"reported_availability_percent", "confirmed_served_share_percent", "coverage_percent", "source_minutes", "generator_hours",
	"days", "days_reported", "days_without_report", "late_reports", "corrections", "data_quality", "unconfirmed_share_percent",
	"incidents_started", "availability_basis", "formula", "formula_version", "generated_at_utc", "assumptions",
}

// exportMonthly godoc
//
//	@Summary		Monthly performance pack (CSV)
//	@Description	One row per site for a calendar month (the current month runs to today): the pooled figures, the minutes assumed rather than confirmed, minutes by source, generator hours, how many days were reported, late reports and corrections, the data-quality level and the incidents that started. Every row says how it was reached. Needs export access and is recorded in the audit log.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			month	query		string	true	"Month, 2026-12"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{file}		file
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/monthly-performance.csv [get]
func (a *api) exportMonthly(c *gin.Context) {
	month, err := time.Parse("2006-01", c.Query("month"))
	if err != nil {
		a.fail(c, badRequest("the month must look like 2026-12"))
		return
	}
	pack, err := a.exports.MonthlyPerformance(c.Request.Context(), actorFrom(c), month, strings.TrimSpace(c.Query("region")))
	if err != nil {
		a.fail(c, err)
		return
	}
	gen := pack.GeneratedAt.UTC().Format(time.RFC3339)
	rows := make([][]string, 0, len(pack.Rows))
	for _, r := range pack.Rows {
		s := r.Summary
		basis := "reported only"
		if r.Site.DefaultAvailable() {
			basis = "available by default"
		}
		rows = append(rows, []string{
			r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name), safeCell(r.Site.Region),
			safeCell(r.Site.ResponsibleTeam), string(r.Site.Status),
			pack.From.Format("2006-01-02"), pack.To.Format("2006-01-02"), pack.Timezone,
			itoa(s.EligibleMinutes), itoa(s.ServedMinutes), itoa(s.OutageMinutes), itoa(s.UnknownMinutes), itoa(s.AssumedMinutes),
			percent(s.ReportedAvailability), percent(s.ConfirmedServedShare), percent(s.Coverage), sourceMinutesCell(s.SourceMinutes), hours1(r.GeneratorHours),
			itoa(r.Quality.Days), itoa(r.Quality.ReportedDays), itoa(r.Quality.Days - r.Quality.ReportedDays), itoa(r.Quality.LateReports),
			itoa(r.Quality.Corrections), string(r.Quality.Level), strconv.FormatFloat(r.Quality.UnconfirmedShare, 'f', 1, 64),
			itoa(r.IncidentsStarted), basis, pack.Formula, pack.Version, gen, services.ExportAssumptions,
		})
	}
	writeCSV(c, fmt.Sprintf("coloplus-monthly-performance_%s.csv", pack.From.Format("2006-01")), monthlyHeader, rows)
}

var weeklyHeader = []string{
	"site_id", "airtel_code", "coloplus_code", "site_name", "region", "responsible_team", "week_from", "week_to", "time_zone",
	"exceptions", "outage_minutes", "outage_events", "days_without_report", "late_reports", "corrections",
	"open_incidents", "overdue_incidents", "open_disputes", "sites_in_view", "generated_at_utc", "note",
}

// exportWeekly godoc
//
//	@Summary		Weekly exception report (CSV)
//	@Description	For the week starting on the given Monday (the current week runs to today), only the sites with something to look at: an outage, a day with no report, a late report, a correction, or an open or overdue incident or dispute. A site with nothing to report is left out and counted in sites_in_view, so a short list is not mistaken for a short portfolio. The open counts are as of when the report is made. Needs export access and is audited.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			week_start	query		string	true	"The Monday, 2026-12-07"
//	@Param			region		query		string	false	"Only sites in this region"
//	@Success		200			{file}		file
//	@Failure		400			{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/weekly-exceptions.csv [get]
func (a *api) exportWeekly(c *gin.Context) {
	start, err := services.ParseDate(c.Query("week_start"))
	if err != nil {
		a.fail(c, err)
		return
	}
	rep, err := a.exports.WeeklyExceptions(c.Request.Context(), actorFrom(c), start, strings.TrimSpace(c.Query("region")))
	if err != nil {
		a.fail(c, err)
		return
	}
	gen := rep.GeneratedAt.UTC().Format(time.RFC3339)
	rows := make([][]string, 0, len(rep.Rows))
	for _, r := range rep.Rows {
		rows = append(rows, []string{
			r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name), safeCell(r.Site.Region),
			safeCell(r.Site.ResponsibleTeam), rep.From.Format("2006-01-02"), rep.To.Format("2006-01-02"), rep.Timezone,
			strings.Join(r.Reasons, ";"), itoa(r.OutageMinutes), itoa(r.OutageEvents), itoa(r.DaysWithoutReport), itoa(r.LateReports),
			itoa(r.Corrections), itoa(r.OpenIncidents), itoa(r.OverdueIncidents), itoa(r.OpenDisputes), itoa(rep.SitesInView), gen,
			"Days without a report are assumed available for sites available by default; they are listed here so they are seen.",
		})
	}
	writeCSV(c, fmt.Sprintf("coloplus-weekly-exceptions_%s.csv", rep.From.Format("2006-01-02")), weeklyHeader, rows)
}

var backlogHeader = []string{
	"site_id", "airtel_code", "coloplus_code", "site_name", "region", "site_status", "time_zone",
	"acceptance_stage", "waiting_on", "next_action", "due_at_utc", "overdue", "hours_since_ready_for_inspection",
	"generated_at_utc", "note",
}

// exportBacklog godoc
//
//	@Summary		Commissioning backlog (CSV)
//	@Description	The sites not yet accepted, with the stage, whose move it is, the next action and when it is due. Time since ready for inspection is reported apart from operational performance. Needs export access and acceptance access; audited.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{file}		file
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/commissioning-backlog.csv [get]
func (a *api) exportBacklog(c *gin.Context) {
	b, err := a.exports.CommissioningBacklog(c.Request.Context(), actorFrom(c), strings.TrimSpace(c.Query("region")))
	if err != nil {
		a.fail(c, err)
		return
	}
	gen := b.GeneratedAt.UTC().Format(time.RFC3339)
	rows := make([][]string, 0, len(b.Rows))
	for _, r := range b.Rows {
		rows = append(rows, []string{
			r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name), safeCell(r.Site.Region),
			string(r.Site.Status), b.Timezone, string(r.Acceptance.Stage), string(r.Acceptance.Responsible),
			safeCell(r.Acceptance.NextAction), utcTime(r.Acceptance.DueAt), yesNo(r.Overdue), itoa(r.ReadyAgeHours), gen,
			"Time waiting for inspection is not operational performance and is not part of availability.",
		})
	}
	writeCSV(c, "coloplus-commissioning-backlog_"+b.GeneratedAt.In(a.loc).Format("2006-01-02")+".csv", backlogHeader, rows)
}
