package httpapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

var dailySummaryHeader = []string{
	"site_id", "airtel_code", "coloplus_code", "site_name", "site_status",
	"date", "time_zone", "service_window",
	"report_status", "report_id", "report_revision", "approved_at_utc", "is_correction",
	"eligible_minutes", "served_minutes", "outage_minutes", "unknown_minutes",
	"coverage_percent", "reported_availability_percent", "confirmed_served_share_percent", "assumed_available_minutes", "assumed_outage_minutes", "excluded_maintenance_minutes", "source_minutes", "provisional",
	"basis", "declared_intervals_ignored",
	"open_disputes", "total_disputes",
	"formula", "formula_version", "generated_at_utc", "assumptions",
}

// exportDailySummaries godoc
//
//	@Summary		Export daily summaries as CSV
//	@Description	One row per site and reporting day, approved reports only. A day without an approved report is still a row, marked no_approved_report with no served or outage figures. For a site that is available by default its availability is shown as assumed (assumed_available_minutes), never as confirmed. Every row carries the report revision, basis, coverage, formula and version, dispute state, time zone, units (minutes and percent), generation time and assumptions. Needs export access; an export is recorded in the audit log.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			from	query		string	true	"First day, 2026-12-01"
//	@Param			to		query		string	true	"Last day (not in the future); at most 92 days"
//	@Param			site_id	query		string	false	"One site; default every site in scope"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{file}		file
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/daily-summaries.csv [get]
func (a *api) exportDailySummaries(c *gin.Context) {
	in := services.ExportDailySummariesInput{SiteID: c.Query("site_id"), Region: strings.TrimSpace(c.Query("region"))}
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
	w := &csvSink{}

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
		} else {
			// No approved report: nothing is confirmed, but a site that is
			// available by default still has an assumed figure, marked as such.
			availability = percent(sum.ReportedAvailability)
		}
		_ = w.Write([]string{
			r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name), string(r.Site.Status),
			r.Date.Format("2006-01-02"), exp.Timezone,
			fmt.Sprintf("%02d:%02d-%02d:%02d", r.Site.ServiceStartMinute/60, r.Site.ServiceStartMinute%60, r.Site.ServiceEndMinute/60, r.Site.ServiceEndMinute%60),
			status, reportID, revision, approvedAt, correction,
			strconv.Itoa(sum.EligibleMinutes), served, outage, strconv.Itoa(sum.UnknownMinutes),
			coverage, availability, confirmed, strconv.Itoa(sum.AssumedMinutes), strconv.Itoa(sum.CarriedMinutes), strconv.Itoa(sum.ExcludedMinutes), sourceMinutesCell(sum.SourceMinutes), yesNo(sum.Provisional()),
			"observed intervals only", strconv.Itoa(r.DeclaredIgnored),
			strconv.Itoa(r.Disputes.Open), strconv.Itoa(r.Disputes.Total),
			exp.Formula, exp.Version, generated, services.ExportAssumptions,
		})
	}
	a.issueCSV(c, "daily_summaries", name, map[string]any{
		"from": exp.From.Format("2006-01-02"), "to": exp.To.Format("2006-01-02"), "site_id": in.SiteID, "region": in.Region,
	}, dailySummaryHeader, w.rows)
}

var incidentHeader = []string{
	"incident_ref", "incident_id", "site_id", "airtel_code", "coloplus_code", "site_name", "region", "responsible_team",
	"category", "severity", "impact", "title", "status", "customer_published",
	"started_at_utc", "restoration_target_utc", "restored_at_utc", "restored_verified", "closed_at_utc",
	"duration_minutes", "time_zone", "generated_at_utc", "assumptions",
}

// exportIncidents godoc
//
//	@Summary		Export the incident register as CSV
//	@Description	One row per reported fault that started in the period, with its site, severity, impact, status and observed times. Airtel gets published incidents only, as everywhere else. Duration is blank until restoration is recorded; a restoration target is a plan and is shown apart. Faults with no outage are told apart by impact. Needs export access; an export is recorded in the audit log.
//	@Tags			exports
//	@Produce		text/csv
//	@Param			from	query		string	true	"First day, 2026-12-01"
//	@Param			to		query		string	true	"Last day (not in the future); at most 92 days"
//	@Param			site_id	query		string	false	"One site; default every site in scope"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Success		200		{file}		file
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/exports/incidents.csv [get]
func (a *api) exportIncidents(c *gin.Context) {
	in := services.ExportIncidentsInput{SiteID: c.Query("site_id"), Region: strings.TrimSpace(c.Query("region"))}
	var err error
	if in.From, err = services.ParseDate(c.Query("from")); err != nil {
		a.fail(c, err)
		return
	}
	if in.To, err = services.ParseDate(c.Query("to")); err != nil {
		a.fail(c, err)
		return
	}
	exp, err := a.exports.IncidentRegister(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}

	name := fmt.Sprintf("coloplus-incidents_%s_%s.csv", exp.From.Format("2006-01-02"), exp.To.Format("2006-01-02"))
	w := &csvSink{}

	utc := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	generated := exp.GeneratedAt.UTC().Format(time.RFC3339)
	for _, r := range exp.Rows {
		i := r.Incident
		// Restoration is an observed time; until it is recorded there is no
		// duration, not a running clock and not zero.
		duration := ""
		if i.RestoredAt != nil {
			duration = strconv.Itoa(int(i.RestoredAt.Sub(i.StartedAt).Minutes()))
		}
		_ = w.Write([]string{
			incidentRef(i.Number), i.ID, r.Site.ID, safeCell(r.Site.AirtelCode), safeCell(r.Site.ColoplusCode), safeCell(r.Site.Name),
			safeCell(r.Site.Region), safeCell(r.Site.ResponsibleTeam),
			string(i.Category), string(i.Severity), string(i.Impact), safeCell(i.Title), string(i.Status), yesNo(i.PublishedAt != nil),
			utc(&i.StartedAt), utc(i.RestorationTarget), utc(i.RestoredAt), yesNo(i.VerifiedAt != nil), utc(i.ClosedAt),
			duration, exp.Timezone, generated, services.ExportIncidentRegisterAssumptions,
		})
	}
	a.issueCSV(c, "incident_register", name, map[string]any{
		"from": exp.From.Format("2006-01-02"), "to": exp.To.Format("2006-01-02"), "site_id": in.SiteID, "region": in.Region,
	}, incidentHeader, w.rows)
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

// sourceMinutesCell writes "generator:600;grid:700", sorted, or nothing.
func sourceMinutesCell(m map[domain.PowerSource]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ":" + strconv.Itoa(m[domain.PowerSource(k)])
	}
	return strings.Join(parts, ";")
}
