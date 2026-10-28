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
	c.JSON(http.StatusCreated, toSiteDTO(created))
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
//	@Param		limit	query		int		false	"Page size (default 50, max 200)"
//	@Param		offset	query		int		false	"Offset"
//	@Success	200		{object}	dto.SiteListResponse
//	@Router		/api/v1/sites [get]
func (a *api) listSites(c *gin.Context) {
	in := services.ListSitesInput{
		Query: c.Query("q"), State: c.Query("state"), Region: c.Query("region"),
		Status: domain.SiteStatus(c.Query("status")), Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
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
	c.JSON(http.StatusOK, dto.SiteListResponse{Sites: out, Total: total, Limit: limit, Offset: offset})
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
	c.JSON(http.StatusOK, toSiteDTO(site))
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
	c.JSON(http.StatusOK, toSiteDTO(updated))
}
