package dto

import "time"

// MaintenanceRequest asks for a planned maintenance window.
type MaintenanceRequest struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Reason   string    `json:"reason"`
}

// MaintenanceDecisionRequest carries the note on a decision. It is required to
// reject or cancel.
type MaintenanceDecisionRequest struct {
	Note string `json:"note"`
}

// MaintenanceDTO is a planned maintenance window. While approved, its time is
// taken out of the site's eligible time.
type MaintenanceDTO struct {
	ID           string     `json:"id"`
	SiteID       string     `json:"site_id"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       time.Time  `json:"ends_at"`
	Minutes      int        `json:"minutes"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status" example:"approved"`
	RequestedBy  string     `json:"requested_by,omitempty"`
	RequestedAt  time.Time  `json:"requested_at"`
	DecidedBy    string     `json:"decided_by,omitempty"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	DecisionNote string     `json:"decision_note,omitempty"`
}

// MaintenanceRulesDTO are the limits a request is held to.
type MaintenanceRulesDTO struct {
	NoticeHours     int `json:"notice_hours"`
	MaxHours        int `json:"max_hours"`
	MonthlyCapHours int `json:"monthly_cap_hours"`
}

type MaintenanceListResponse struct {
	Windows []MaintenanceDTO    `json:"windows"`
	Rules   MaintenanceRulesDTO `json:"rules"`
	Total   int                 `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}
