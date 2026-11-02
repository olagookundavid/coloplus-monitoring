package httpapi

import (
	"net/http"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

const fuelExplanation = "An estimate, not a meter reading. It starts from the latest tank reading (or the first delivery, " +
	"taking the tank as empty before it), adds deliveries, and takes off the site's agreed litres per hour for each hour " +
	"the generator was reported running. A day with no approved generator hours is unknown, not zero, so the level is " +
	"then the most the tank could hold."

func toFuelEntryDTO(e domain.FuelEntry, staff bool) dto.FuelEntryDTO {
	out := dto.FuelEntryDTO{
		ID: e.ID, SiteID: e.SiteID, Kind: string(e.Kind), Litres: e.Litres, OccurredAt: e.OccurredAt, CreatedAt: e.CreatedAt,
	}
	if staff {
		out.Note, out.RecordedBy, out.Voided, out.VoidReason = e.Note, e.RecordedBy, e.Voided(), e.VoidReason
	}
	return out
}

func toFuelEstimateDTO(e domain.FuelEstimate, site domain.Site) dto.FuelEstimateDTO {
	return dto.FuelEstimateDTO{
		Basis: "estimate", Available: e.Available, Reason: e.Reason, Litres: e.Litres, HoursRemaining: e.HoursRemaining,
		ConsumptionLPH: e.ConsumptionLPH, AnchorKind: string(e.AnchorKind), AnchorAt: e.AnchorAt,
		AssumedEmptyStart: e.AssumedEmptyStart, HoursRun: e.HoursRun, ConsumedLitres: e.ConsumedLitres,
		UnreportedDays: e.UnreportedDays, Provisional: e.Provisional, EstimatedEmpty: e.EstimatedEmpty,
		OverCapacity: e.OverCapacity, TankCapacityLitres: site.TankCapacityLitres, AsOf: e.AsOf,
		Formula: e.Formula, FormulaVersion: e.Version, Explanation: fuelExplanation,
	}
}

// recordFuel godoc
//
//	@Summary		Record a fuel delivery or a tank reading
//	@Description	delivery adds litres to the tank ("1,000 litres delivered"). reading says what the tank held when someone looked and replaces the estimate. The estimate is then worked out from the site's agreed litres per hour and the generator hours reported each day. Send client_key and retry safely.
//	@Tags			fuel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Site id"
//	@Param			body	body		dto.RecordFuelRequest	true	"Entry"
//	@Success		201		{object}	dto.FuelEntryDTO
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/fuel [post]
func (a *api) recordFuel(c *gin.Context) {
	var req dto.RecordFuelRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	e, err := a.fuel.Record(c.Request.Context(), actor, c.Param("id"), services.FuelInput{
		Kind: domain.FuelKind(req.Kind), Litres: req.Litres, OccurredAt: req.OccurredAt, Note: req.Note, ClientKey: req.ClientKey,
	})
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toFuelEntryDTO(e, a.fuel.IsStaff(actor)))
}

// fuelStatus godoc
//
//	@Summary		A site's fuel: estimated level and entries
//	@Description	The estimate is labelled as one and says how it was reached. With no agreed rate, or nothing recorded, available is false and reason says why; there is never a made-up number. Notes and who recorded an entry are for Coloplus only.
//	@Tags			fuel
//	@Produce		json
//	@Param			id	path		string	true	"Site id"
//	@Success		200	{object}	dto.FuelStatusResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/api/v1/sites/{id}/fuel [get]
func (a *api) fuelStatus(c *gin.Context) {
	actor := actorFrom(c)
	v, err := a.fuel.Status(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		a.fail(c, err)
		return
	}
	staff := a.fuel.IsStaff(actor)
	resp := dto.FuelStatusResponse{SiteID: v.Site.ID, Estimate: toFuelEstimateDTO(v.Estimate, v.Site), Entries: make([]dto.FuelEntryDTO, 0, len(v.Entries))}
	for _, e := range v.Entries {
		resp.Entries = append(resp.Entries, toFuelEntryDTO(e, staff))
	}
	c.JSON(http.StatusOK, resp)
}

// voidFuel godoc
//
//	@Summary		Void a mistaken fuel entry
//	@Description	The entry stays in the record with the reason and no longer counts. Repeating it changes nothing.
//	@Tags			fuel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Fuel entry id"
//	@Param			body	body		dto.VoidFuelRequest	true	"Reason"
//	@Success		200		{object}	dto.FuelEntryDTO
//	@Router			/api/v1/fuel/{id}/void [post]
func (a *api) voidFuel(c *gin.Context) {
	var req dto.VoidFuelRequest
	if err := bindJSON(c, &req); err != nil {
		a.fail(c, err)
		return
	}
	actor := actorFrom(c)
	e, err := a.fuel.Void(c.Request.Context(), actor, c.Param("id"), req.Reason)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toFuelEntryDTO(e, a.fuel.IsStaff(actor)))
}
