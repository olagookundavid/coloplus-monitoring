package httpapi

import (
	"net/http"

	"coloplus-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

// listOutages godoc
//
//	@Summary		The outages behind the dashboard figures
//	@Description	Every reported outage in the period from the approved reports, newest first, each merged so no minute is counted twice, with the report it came from. This is what a click on an outage card opens. The reason is for Coloplus staff only. Give from and to (at most 92 days, none in the future); region and site_id narrow it.
//	@Tags			dashboard
//	@Produce		json
//	@Param			from	query		string	true	"First day"
//	@Param			to		query		string	true	"Last day"
//	@Param			region	query		string	false	"Only sites in this region"
//	@Param			site_id	query		string	false	"Only this site"
//	@Param			limit	query		int		false	"Page size (default 100, max 500)"
//	@Param			offset	query		int		false	"Offset"
//	@Success		200		{object}	dto.OutageListResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/dashboard/outages [get]
func (a *api) listOutages(c *gin.Context) {
	p, err := a.dashboardPeriod(c, false)
	if err != nil {
		a.fail(c, err)
		return
	}
	l, err := a.dashboard.Outages(c.Request.Context(), actorFrom(c), p, c.Query("site_id"), intQuery(c, "limit"), intQuery(c, "offset"))
	if err != nil {
		a.fail(c, err)
		return
	}
	resp := dto.OutageListResponse{
		From: l.Period.From.Format("2006-01-02"), To: l.Period.To.Format("2006-01-02"), Timezone: a.loc.String(),
		GeneratedAt: l.GeneratedAt, TotalMinutes: l.TotalMinutes, Total: l.Total, Limit: l.Limit, Offset: l.Offset,
		Outages: make([]dto.OutageItemDTO, 0, len(l.Items)),
	}
	for _, it := range l.Items {
		resp.Outages = append(resp.Outages, dto.OutageItemDTO{
			SiteID: it.Site.ID, AirtelCode: it.Site.AirtelCode, ColoplusCode: it.Site.ColoplusCode, SiteName: it.Site.Name,
			Region: it.Site.Region, Start: it.Start.In(a.loc), End: it.End.In(a.loc), Minutes: it.Minutes,
			ReportID: it.ReportID, ReportDate: it.ReportDate.Format("2006-01-02"), ReportRevision: it.Revision, Reason: it.Reason,
		})
	}
	a.respondNamed(c, http.StatusOK, resp)
}

// incidentOutages godoc
//
//	@Summary		The reported outage during an incident
//	@Description	Links the incident to the outage the approved daily reports show between its start and its restoration (or now, while it is open), each minute counted once. Nothing is added to the daily figures. An "Airtel" user sees this only for a published incident.
//	@Tags			incidents
//	@Produce		json
//	@Param			id	path		string	true	"Incident id"
//	@Success		200	{object}	dto.IncidentOutagesResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/incidents/{id}/outages [get]
func (a *api) incidentOutages(c *gin.Context) {
	o, err := a.dashboard.IncidentOutages(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	resp := dto.IncidentOutagesResponse{
		IncidentID: o.IncidentID, From: o.From.In(a.loc), To: o.To.In(a.loc), Open: o.Open, TotalMinutes: o.TotalMinutes,
		Outages: make([]dto.IncidentOutageDTO, 0, len(o.Items)),
	}
	for _, it := range o.Items {
		resp.Outages = append(resp.Outages, dto.IncidentOutageDTO{
			Start: it.Start.In(a.loc), End: it.End.In(a.loc), Minutes: it.Minutes,
			ReportID: it.ReportID, ReportDate: it.ReportDate.Format("2006-01-02"), ReportRevision: it.Revision,
		})
	}
	a.respondNamed(c, http.StatusOK, resp)
}
