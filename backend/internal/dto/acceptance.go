package dto

import "time"

type RequestAcceptanceRequest struct {
	NextAction string     `json:"next_action,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
}

type TransitionAcceptanceRequest struct {
	// Stage is built, ready_for_inspection, inspected, accepted, rejected or
	// on_hold. To resume from a hold, name the stage it left.
	Stage string `json:"stage" example:"ready_for_inspection"`
	// Note is the dossier when declaring the site ready, the reason when
	// rejecting or holding, and what was fixed when rework follows a rejection.
	Note string `json:"note,omitempty"`
	// NextAction and DueAt set the plan after this step.
	NextAction string     `json:"next_action,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
}

type AcceptanceDTO struct {
	ID     string `json:"id"`
	SiteID string `json:"site_id"`
	Stage  string `json:"stage"`
	// HeldFrom is the stage a hold returns to.
	HeldFrom string `json:"held_from,omitempty"`
	// WaitingOn is coloplus or airtel: whose move it is.
	WaitingOn  string     `json:"waiting_on"`
	NextAction string     `json:"next_action,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Overdue    bool       `json:"overdue"`
	// Dossier is what Coloplus submitted when it declared the site ready.
	Dossier    string `json:"dossier,omitempty"`
	Reason     string `json:"reason,omitempty"`
	HoldReason string `json:"hold_reason,omitempty"`
	// ReadyAgeHours is how long the site has waited for inspection since it
	// was declared ready. It is not operational performance.
	ReadyAgeHours int        `json:"ready_age_hours"`
	RequestedAt   time.Time  `json:"requested_at"`
	BuiltAt       *time.Time `json:"built_at,omitempty"`
	ReadyAt       *time.Time `json:"ready_at,omitempty"`
	InspectedAt   *time.Time `json:"inspected_at,omitempty"`
	// DecidedAt is when Airtel accepted. Nothing else counts as acceptance.
	DecidedAt *time.Time `json:"decided_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type AcceptanceEventDTO struct {
	ID        string    `json:"id"`
	Party     string    `json:"party"`
	FromStage string    `json:"from_stage,omitempty"`
	ToStage   string    `json:"to_stage"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AcceptanceDetailResponse struct {
	Acceptance AcceptanceDTO        `json:"acceptance"`
	Events     []AcceptanceEventDTO `json:"events"`
}

type AcceptanceListResponse struct {
	Acceptances []AcceptanceDTO `json:"acceptances"`
	Total       int             `json:"total"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
}
