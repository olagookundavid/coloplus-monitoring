package dto

import "time"

// CalculationRunRequest asks for a month to be worked out and frozen.
type CalculationRunRequest struct {
	Month  string `json:"month" example:"2026-10"`
	Region string `json:"region,omitempty"`
	Label  string `json:"label,omitempty" example:"October close"`
}

// RunRowDTO is one site's frozen figures. Percentages are null where they did
// not apply, never zero; assumed_minutes is time counted as available only
// because nobody reported it.
type RunRowDTO struct {
	SiteID               string   `json:"site_id"`
	AirtelCode           string   `json:"airtel_code"`
	SiteName             string   `json:"site_name"`
	EligibleMinutes      int      `json:"eligible_minutes"`
	ServedMinutes        int      `json:"served_minutes"`
	OutageMinutes        int      `json:"outage_minutes"`
	UnknownMinutes       int      `json:"unknown_minutes"`
	AssumedMinutes       int      `json:"assumed_minutes"`
	DaysInPeriod         int      `json:"days_in_period"`
	DaysReported         int      `json:"days_reported"`
	ReportedAvailability *float64 `json:"reported_availability_percent"`
	ConfirmedServedShare *float64 `json:"confirmed_served_share_percent"`
	Coverage             *float64 `json:"coverage_percent"`
	GeneratorHours       float64  `json:"generator_hours"`
	IncidentsStarted     int      `json:"incidents_started"`
	ReportsFingerprint   string   `json:"reports_fingerprint"`
}

// CalculationRunDTO is a stored run. rows is left out of the list.
type CalculationRunDTO struct {
	ID             string      `json:"id"`
	Kind           string      `json:"kind" example:"monthly_performance"`
	PeriodFrom     string      `json:"period_from" example:"2026-10-01"`
	PeriodTo       string      `json:"period_to" example:"2026-10-31"`
	Region         string      `json:"region"`
	Label          string      `json:"label"`
	Formula        string      `json:"formula"`
	FormulaVersion string      `json:"formula_version"`
	SiteCount      int         `json:"site_count"`
	CreatedBy      string      `json:"created_by"`
	CreatedAt      time.Time   `json:"created_at"`
	Rows           []RunRowDTO `json:"rows,omitempty"`
}

type CalculationRunListResponse struct {
	Runs   []CalculationRunDTO `json:"runs"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// RunChangeDTO is a site whose figures differ between the run and now.
type RunChangeDTO struct {
	SiteID     string     `json:"site_id"`
	AirtelCode string     `json:"airtel_code"`
	SiteName   string     `json:"site_name"`
	Then       *RunRowDTO `json:"then,omitempty"`
	Now        *RunRowDTO `json:"now,omitempty"`
	// ReportsChanged: the approved reports behind the figures are not the same
	// ones any more (a correction, or a report approved after the run).
	ReportsChanged bool `json:"reports_changed"`
}

// RunComparisonDTO sets a stored run against the same period worked out now.
type RunComparisonDTO struct {
	Run            CalculationRunDTO `json:"run"`
	GeneratedAt    time.Time         `json:"generated_at"`
	FormulaChanged bool              `json:"formula_changed"`
	CurrentVersion string            `json:"current_formula_version"`
	Unchanged      int               `json:"unchanged"`
	Changes        []RunChangeDTO    `json:"changes"`
}
