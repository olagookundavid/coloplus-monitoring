package dto

import "time"

// RequestCorrectionRequest asks to correct a single incident field. There is
// no original_value: the service snapshots the current value itself, inside
// the transaction, never from the client.
type RequestCorrectionRequest struct {
	// Field is started_at or restored_at.
	Field          string    `json:"field" example:"restored_at"`
	CorrectedValue time.Time `json:"corrected_value"`
	Reason         string    `json:"reason"`
}

type RejectCorrectionRequest struct {
	Note string `json:"note"`
}

type IncidentCorrectionDTO struct {
	ID              string     `json:"id"`
	Ref             string     `json:"ref" example:"COR-000058"`
	IncidentID      string     `json:"incident_id"`
	IncidentLabel   string     `json:"incident_label,omitempty" names:"IncidentID:incident"`
	SiteID          string     `json:"site_id"`
	SiteName        string     `json:"site_name,omitempty" names:"SiteID:site"`
	Field           string     `json:"field"`
	OriginalValue   time.Time  `json:"original_value"`
	CorrectedValue  time.Time  `json:"corrected_value"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	RequestedBy     string     `json:"requested_by"`
	RequestedByName string     `json:"requested_by_name,omitempty" names:"RequestedBy:user"`
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	ReviewedByName  string     `json:"reviewed_by_name,omitempty" names:"ReviewedBy:user"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote      string     `json:"review_note,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type IncidentCorrectionListResponse struct {
	Corrections []IncidentCorrectionDTO `json:"corrections"`
	Total       int                     `json:"total"`
	Limit       int                     `json:"limit"`
	Offset      int                     `json:"offset"`
}
