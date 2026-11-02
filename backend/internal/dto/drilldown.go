package dto

import "time"

// OutageItemDTO is one stretch of reported outage behind a dashboard figure.
type OutageItemDTO struct {
	SiteID       string    `json:"site_id"`
	AirtelCode   string    `json:"airtel_code"`
	ColoplusCode string    `json:"coloplus_code"`
	SiteName     string    `json:"site_name"`
	Region       string    `json:"region"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Minutes      int       `json:"minutes"`
	// The approved report it came from, so the figure can be traced.
	ReportID       string `json:"report_id"`
	ReportDate     string `json:"report_date"`
	ReportRevision int    `json:"report_revision"`
	// Reason is why the site was unavailable; sent to Coloplus staff only.
	Reason string `json:"reason,omitempty"`
}

type OutageListResponse struct {
	From        string    `json:"from"`
	To          string    `json:"to"`
	Timezone    string    `json:"timezone"`
	GeneratedAt time.Time `json:"generated_at"`
	// TotalMinutes and Total cover every outage in the period, not just this page.
	TotalMinutes int             `json:"total_minutes"`
	Total        int             `json:"total"`
	Limit        int             `json:"limit"`
	Offset       int             `json:"offset"`
	Outages      []OutageItemDTO `json:"outages"`
}

type IncidentOutageDTO struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	// Minutes is the part of it inside the incident's window.
	Minutes        int    `json:"minutes"`
	ReportID       string `json:"report_id"`
	ReportDate     string `json:"report_date"`
	ReportRevision int    `json:"report_revision"`
}

// IncidentOutagesResponse links an incident to the outage the daily reports show
// during it. Nothing here is added to the daily figures a second time.
type IncidentOutagesResponse struct {
	IncidentID string    `json:"incident_id"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	// Open: the incident is not restored yet, so the window runs to now.
	Open bool `json:"open"`
	// TotalMinutes is the outage inside the window, each minute counted once.
	TotalMinutes int                 `json:"total_minutes"`
	Outages      []IncidentOutageDTO `json:"outages"`
}
