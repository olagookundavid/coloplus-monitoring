package dto

import "time"

type SiteDTO struct {
	ID                 string    `json:"id"`
	AirtelCode         string    `json:"airtel_code"`
	ColoplusCode       string    `json:"coloplus_code"`
	Name               string    `json:"name"`
	Address            string    `json:"address"`
	Region             string    `json:"region"`
	State              string    `json:"state"`
	SiteType           string    `json:"site_type"`
	PowerSources       []string  `json:"power_sources" example:"grid,generator"`
	ServiceStartMinute int       `json:"service_start_minute" example:"0"`
	ServiceEndMinute   int       `json:"service_end_minute" example:"1440"`
	FreshnessHours     int       `json:"freshness_hours" example:"30"`
	Status             string    `json:"status" example:"active"`
	CommissioningDate  *string   `json:"commissioning_date,omitempty" example:"2026-09-01"`
	Latitude           *float64  `json:"latitude,omitempty"`
	Longitude          *float64  `json:"longitude,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SiteRequest creates or replaces a site. Leave service window and freshness
// at zero to get a 24-hour window and a 30-hour freshness threshold.
type SiteRequest struct {
	AirtelCode         string   `json:"airtel_code"`
	ColoplusCode       string   `json:"coloplus_code"`
	Name               string   `json:"name"`
	Address            string   `json:"address"`
	Region             string   `json:"region"`
	State              string   `json:"state"`
	SiteType           string   `json:"site_type"`
	PowerSources       []string `json:"power_sources" example:"grid,generator"`
	ServiceStartMinute int      `json:"service_start_minute"`
	ServiceEndMinute   int      `json:"service_end_minute"`
	FreshnessHours     int      `json:"freshness_hours"`
	Status             string   `json:"status" example:"pending_acceptance"`
	CommissioningDate  *string  `json:"commissioning_date,omitempty" example:"2026-09-01"`
	Latitude           *float64 `json:"latitude,omitempty"`
	Longitude          *float64 `json:"longitude,omitempty"`
}

type SiteListResponse struct {
	Sites  []SiteDTO `json:"sites"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// SiteDayDTO is one reporting day on the site screen. A day without an
// approved report has no summary: a gap, never zero and never available.
type SiteDayDTO struct {
	Date         string      `json:"date" example:"2026-12-01"`
	ReportID     string      `json:"report_id,omitempty"`
	Revision     int         `json:"revision,omitempty"`
	Summary      *SummaryDTO `json:"summary,omitempty"`
	OpenDisputes int         `json:"open_disputes,omitempty"`
}

// SiteOverviewResponse is the whole site screen in one call. Sections the
// caller may not see are left out. Airtel gets published incidents only.
type SiteOverviewResponse struct {
	Site     SiteDTO `json:"site"`
	Timezone string  `json:"timezone"`
	// Condition is available, outage_reported or awaiting_update.
	Condition string     `json:"condition"`
	AsOf      *time.Time `json:"as_of,omitempty"`
	// Days runs newest first and includes days with no approved report.
	Days []SiteDayDTO `json:"days"`

	Incidents          []IncidentDTO          `json:"incidents,omitempty"`
	PublishedIncidents []PublishedIncidentDTO `json:"published_incidents,omitempty"`
	OpenIncidents      *int                   `json:"open_incidents,omitempty"`
	Disputes           []DisputeDTO           `json:"disputes,omitempty"`
	OpenDisputes       *int                   `json:"open_disputes,omitempty"`
	Acceptance         *AcceptanceDTO         `json:"acceptance,omitempty"`
}
