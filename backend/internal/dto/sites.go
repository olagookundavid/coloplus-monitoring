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
