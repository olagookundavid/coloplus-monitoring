package httpapi

import (
	"net/http"
	"strconv"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func intQuery(c *gin.Context, key string) int {
	n, _ := strconv.Atoi(c.Query(key))
	return n
}

// createSite godoc
//
//	@Summary	Register a site
//	@Tags		sites
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.SiteRequest	true	"Site"
//	@Success	201		{object}	dto.SiteDTO
//	@Failure	400		{object}	dto.ErrorResponse
//	@Failure	409		{object}	dto.ErrorResponse
//	@Router		/api/v1/sites [post]
func (a *api) createSite(c *gin.Context) {
	var req dto.SiteRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	site, err := fromSiteRequest(req)
	if err != nil {
		a.fail(c, err)
		return
	}
	created, err := a.sites.Create(c.Request.Context(), actorFrom(c), site)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusCreated, toSiteDTO(created))
}

// listSites godoc
//
//	@Summary	List sites in your scope
//	@Tags		sites
//	@Produce	json
//	@Param		q		query		string	false	"Search Airtel code, Coloplus code or name"
//	@Param		state	query		string	false	"State"
//	@Param		region	query		string	false	"Region"
//	@Param		status	query		string	false	"pending_acceptance, active or suspended"
//	@Param		source	query		string	false	"Installed power source: grid, generator, battery, solar or building_supply"
//	@Param		acceptance_stage	query	string	false	"requested, built, ready_for_inspection, inspected, accepted, rejected or on_hold"
//	@Param		assigned_to	query		string	false	"User id; Coloplus staff only"
//	@Param		limit	query		int		false	"Page size (default 50, max 200)"
//	@Param		offset	query		int		false	"Offset"
//	@Success	200		{object}	dto.SiteListResponse
//	@Router		/api/v1/sites [get]
func (a *api) listSites(c *gin.Context) {
	in := services.ListSitesInput{
		Query: c.Query("q"), State: c.Query("state"), Region: c.Query("region"),
		Status: domain.SiteStatus(c.Query("status")), Source: domain.PowerSource(c.Query("source")),
		AcceptanceStage: domain.AcceptanceStage(c.Query("acceptance_stage")), AssignedTo: c.Query("assigned_to"),
		Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	sites, total, err := a.sites.List(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	out := make([]dto.SiteDTO, len(sites))
	for i, s := range sites {
		out[i] = toSiteDTO(s)
	}
	a.respondNamed(c, http.StatusOK, dto.SiteListResponse{Sites: out, Total: total, Limit: limit, Offset: offset})
}

// getSite godoc
//
//	@Summary	One site
//	@Tags		sites
//	@Produce	json
//	@Param		id	path		string	true	"Site id"
//	@Success	200	{object}	dto.SiteDTO
//	@Failure	404	{object}	dto.ErrorResponse
//	@Router		/api/v1/sites/{id} [get]
func (a *api) getSite(c *gin.Context) {
	site, err := a.sites.Get(c.Request.Context(), actorFrom(c), c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toSiteDTO(site))
}

// updateSite godoc
//
//	@Summary	Replace a site's details
//	@Tags		sites
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"Site id"
//	@Param		body	body		dto.SiteRequest	true	"Site"
//	@Success	200		{object}	dto.SiteDTO
//	@Router		/api/v1/sites/{id} [put]
func (a *api) updateSite(c *gin.Context) {
	var req dto.SiteRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	site, err := fromSiteRequest(req)
	if err != nil {
		a.fail(c, err)
		return
	}
	updated, err := a.sites.Update(c.Request.Context(), actorFrom(c), c.Param("id"), site)
	if err != nil {
		a.fail(c, err)
		return
	}
	a.respondNamed(c, http.StatusOK, toSiteDTO(updated))
}

// siteOverview godoc
//
//	@Summary		The whole site screen in one call
//	@Description	Profile, current condition and as-of time, the last days (gaps included, never shown as zero or available), open incidents, open disputes and the acceptance process. Sections the caller may not see are left out; Airtel gets published incidents only.
//	@Tags			sites
//	@Produce		json
//	@Param			id		path		string	true	"Site id"
//	@Param			days	query		int		false	"How many reporting days to show (default 14, max 90)"
//	@Success		200		{object}	dto.SiteOverviewResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/overview [get]
func (a *api) siteOverview(c *gin.Context) {
	actor := actorFrom(c)
	o, err := a.siteOverviews.Overview(c.Request.Context(), actor, c.Param("id"), intQuery(c, "days"))
	if err != nil {
		a.fail(c, err)
		return
	}
	now := a.clock.Now()
	resp := dto.SiteOverviewResponse{
		Site: toSiteDTO(o.Site), Timezone: a.loc.String(), Condition: string(o.Condition), AsOf: o.AsOf,
		Days: make([]dto.SiteDayDTO, 0, len(o.Days)),
	}
	for _, d := range o.Days {
		row := dto.SiteDayDTO{Date: d.Date.Format("2006-01-02"), ReportID: d.ReportID, Revision: d.Revision, OpenDisputes: d.OpenDisputes}
		if d.Summary != nil {
			sum := toSummaryDTO(*d.Summary)
			row.Summary = &sum
		}
		resp.Days = append(resp.Days, row)
	}
	if o.ShowIncidents {
		resp.OpenIncidents = &o.IncidentsTotal
		for _, i := range o.Incidents {
			internal, published := a.incidentBody(actor, i, nil)
			if internal != nil {
				resp.Incidents = append(resp.Incidents, *internal)
			} else {
				resp.PublishedIncidents = append(resp.PublishedIncidents, *published)
			}
		}
	}
	if o.ShowDisputes {
		resp.OpenDisputes = &o.DisputesTotal
		staff := a.disputes.IsStaff(actor)
		for _, d := range o.Disputes {
			resp.Disputes = append(resp.Disputes, toDisputeDTO(d, staff))
		}
	}
	if o.Acceptance != nil {
		acc := toAcceptanceDTO(*o.Acceptance, now)
		resp.Acceptance = &acc
	}
	a.respondNamed(c, http.StatusOK, resp)
}

// importSites godoc
//
//	@Summary		Import sites from CSV
//	@Description	Creates new sites and updates existing ones, matched on the Airtel code. Send the file's text in the csv field (the front end reads the chosen file and sends its text; the API takes JSON only). Columns: airtel_code (required), coloplus_code, name, state, power_sources (grid;generator), region, address, site_type, responsible_team, service_start and service_end (06:00, 24:00), freshness_hours, status, commissioning_date (2026-09-01), latitude, longitude, consumption_lph, tank_capacity_litres, unreported_time, generator_capacity, access_contact_name, access_contact_phone, supervisor_name. On an existing site a blank cell leaves the value as it is. At most 1,000 rows. It is all or nothing: if any row has a problem nothing is written and every problem is listed by line. Set dry_run to check a file first. Needs site management; recorded in the audit log.
//	@Tags			sites
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SiteImportRequest	true	"File text"
//	@Success		200		{object}	dto.SiteImportResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/import [post]
func (a *api) importSites(c *gin.Context) {
	var req dto.SiteImportRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	res, err := a.sites.ImportCSV(c.Request.Context(), actorFrom(c), []byte(req.CSV), req.DryRun)
	if err != nil {
		a.fail(c, err)
		return
	}
	out := dto.SiteImportResponse{
		DryRun: res.DryRun, Applied: res.Applied, Created: res.Created, Updated: res.Updated,
		Unchanged: res.Unchanged, Rejected: res.Rejected, Rows: make([]dto.SiteImportRowDTO, len(res.Rows)),
	}
	for i, r := range res.Rows {
		out.Rows[i] = dto.SiteImportRowDTO{Line: r.Line, AirtelCode: r.AirtelCode, Action: string(r.Action), Problems: r.Problems}
	}
	a.respondNamed(c, http.StatusOK, out)
}
