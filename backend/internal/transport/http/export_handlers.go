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

var dailySummaryHeader = []string{
	"site_id", "airtel_code", "coloplus_code", "site_name", "site_status",
	"date", "time_zone", "service_window",
	"report_status", "report_id", "report_revision", "approved_at_utc", "is_correction",
	"eligible_minutes", "served_minutes", "outage_minutes", "unknown_minutes",
	"coverage_percent", "reported_availability_percent", "confirmed_served_share_percent", "provisional",
	"basis", "declared_intervals_ignored",
	"open_disputes", "total_disputes",
	"formula", "formula_version", "generated_at_utc", "assumptions",
}

// exportDailySummaries godoc
//
//	@Summary		Export daily summaries as CSV
//	@Description	One row per site and reporting day, approved reports only. A day without an approved report is still a row, marked no_approved_report with no served or outage figures, never zero and never available. Every row carries the report revision, basis, coverage, formula and version, dispute state, time zone, units (minutes and percent), generation time and assumptions. Needs export access; an export is recorded in the audit log.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			from	query		string	true	"First day, 2026-12-01"
//	@Param			to		query		string	true	"Last day (not in the future); at most 92 days"
//	@Param			site_id	query		string	false	"One site; default every site in scope"
//	@Success		200		{file}		file
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/daily-summaries.csv [get]
func (a *api) exportDailySummaries(c *gin.Context) {
	in := services.ExportDailySummariesInput{SiteID: c.Query("site_id")}
	var err error
	if in.From, err = services.ParseDate(c.Query("from")); err != nil {
		a.fail(c, err)
		return
	}
	if in.To, err = services.ParseDate(c.Query("to")); err != nil {
		a.fail(c, err)
		return
	}
	exp, err := a.exports.DailySummaries(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}

	name := fmt.Sprintf("coloplus-daily-summaries_%s_%s.csv", exp.From.Format("2006-01-02"), exp.To.Format("2006-01-02"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	// The byte order mark lets Excel read accents correctly.
	_, _ = c.Writer.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(c.Writer)
	_ = w.Write(dailySummaryHeader)

	generated := exp.GeneratedAt.UTC().Format(time.RFC3339)
	for _, r := range exp.Rows {
		sum := r.Summary
		status, reportID, revision, approvedAt, correction := "no_approved_report", "", "", "", ""
		// Without an approved report there are no figures at all: not zero,
		// not a percentage. Only the eligible and unknown minutes are known.
		var served, outage, coverage, availability, confirmed string
		if r.Report != nil {
			status, reportID, revision = "approved", r.Report.ID, strconv.Itoa(r.Report.Revision)
			if r.Report.ReviewedAt != nil {
				approvedAt = r.Report.ReviewedAt.UTC().Format(time.RFC3339)
			}
			correction = yesNo(r.Report.SupersedesID != "")
			served, outage = strconv.Itoa(sum.ServedMinutes), strconv.Itoa(sum.OutageMinutes)
			coverage, availability, confirmed = percent(sum.Coverage), percent(sum.ReportedAvailability), percent(sum.ConfirmedServedShare)
		}
		_ = w.Write([]string{
			r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name), string(r.Site.Status),
			r.Date.Format("2006-01-02"), exp.Timezone,
			fmt.Sprintf("%02d:%02d-%02d:%02d", r.Site.ServiceStartMinute/60, r.Site.ServiceStartMinute%60, r.Site.ServiceEndMinute/60, r.Site.ServiceEndMinute%60),
			status, reportID, revision, approvedAt, correction,
			strconv.Itoa(sum.EligibleMinutes), served, outage, strconv.Itoa(sum.UnknownMinutes),
			coverage, availability, confirmed, yesNo(sum.Provisional()),
			"observed intervals only", strconv.Itoa(r.DeclaredIgnored),
			strconv.Itoa(r.Disputes.Open), strconv.Itoa(r.Disputes.Total),
			exp.Formula, exp.Version, generated, services.ExportAssumptions,
		})
	}
	w.Flush()
}

func percent(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', 2, 64)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// safeCell stops a text value from being run as a formula when the file is
// opened in a spreadsheet.
func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
