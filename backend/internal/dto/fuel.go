package dto

import "time"

type RecordFuelRequest struct {
	// Kind is delivery (litres added to the tank) or reading (litres the tank
	// held when someone looked, which replaces the estimate).
	Kind   string  `json:"kind" example:"delivery"`
	Litres float64 `json:"litres" example:"1000"`
	// OccurredAt is when it happened (RFC 3339); it cannot be in the future.
	OccurredAt time.Time `json:"occurred_at"`
	// Note is for Coloplus only; the customer never sees it.
	Note string `json:"note,omitempty"`
	// ClientKey is optional. Send the same value when retrying a request that
	// may have gone through, and you get the same entry back, not a second one.
	ClientKey string `json:"client_key,omitempty"`
}

type VoidFuelRequest struct {
	Reason string `json:"reason"`
}

// FuelEntryDTO is one delivery or reading. Note, RecordedBy and the void
// details are only sent to Coloplus staff; a customer is never sent a voided entry.
type FuelEntryDTO struct {
	ID         string    `json:"id"`
	SiteID     string    `json:"site_id"`
	Kind       string    `json:"kind"`
	Litres     float64   `json:"litres"`
	OccurredAt time.Time `json:"occurred_at"`
	Note       string    `json:"note,omitempty"`
	RecordedBy string    `json:"recorded_by,omitempty"`
	Voided     bool      `json:"voided,omitempty"`
	VoidReason string    `json:"void_reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// FuelEstimateDTO is the estimated tank level and exactly how it was reached.
// It is an estimate from entries and an agreed rate, not a meter reading.
type FuelEstimateDTO struct {
	// Basis is always "estimate": the level comes from what people entered and
	// the agreed litres per hour.
	Basis     string `json:"basis" example:"estimate"`
	Available bool   `json:"available"`
	// Reason says why there is no estimate, when there is none.
	Reason string `json:"reason,omitempty"`
	// Litres is the estimated level. While UnreportedDays is above 0 it is the
	// most the tank could hold, because those days' use is unknown.
	Litres         *float64 `json:"litres"`
	HoursRemaining *float64 `json:"hours_remaining"`
	// ConsumptionLPH is the agreed litres per running hour used.
	ConsumptionLPH float64 `json:"consumption_lph"`
	// The estimate starts from the latest reading, or the first delivery.
	AnchorKind        string    `json:"anchor_kind,omitempty"`
	AnchorAt          time.Time `json:"anchor_at,omitempty"`
	AssumedEmptyStart bool      `json:"assumed_empty_start"`
	// HoursRun and ConsumedLitres cover the reported days since then.
	HoursRun       float64 `json:"hours_run"`
	ConsumedLitres float64 `json:"consumed_litres"`
	UnreportedDays int     `json:"unreported_days"`
	Provisional    bool    `json:"provisional"`
	EstimatedEmpty bool    `json:"estimated_empty"`
	// OverCapacity means the estimate exceeds the tank, so an entry is probably wrong.
	OverCapacity       bool      `json:"over_capacity"`
	TankCapacityLitres *float64  `json:"tank_capacity_litres"`
	AsOf               time.Time `json:"as_of"`
	Formula            string    `json:"formula" example:"fuel-estimate"`
	FormulaVersion     string    `json:"formula_version" example:"1"`
	Explanation        string    `json:"explanation"`
}

type FuelStatusResponse struct {
	SiteID   string          `json:"site_id"`
	Estimate FuelEstimateDTO `json:"estimate"`
	// Entries are newest first.
	Entries []FuelEntryDTO `json:"entries"`
}
