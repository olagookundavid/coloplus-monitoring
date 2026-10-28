package dto

import "time"

type ReportIncidentRequest struct {
	// Category is grid_loss, building_supply_loss, generator_failure, low_fuel,
	// battery_failure, rectifier_fault, electrical_fault, access_restriction or other.
	Category string `json:"category" example:"generator_failure"`
	// Severity is critical, major or minor.
	Severity string `json:"severity" example:"major"`
	// Impact is outage for a service interruption, no_outage for a fault that caused none.
	Impact      string `json:"impact" example:"outage"`
	Title       string `json:"title" example:"Generator failed to start after grid loss"`
	Description string `json:"description"`
	// StartedAt is when the fault began, as best known.
	StartedAt time.Time `json:"started_at" example:"2026-12-01T06:40:00+01:00"`
	// DetectedAt is when it was noticed; defaults to StartedAt.
	DetectedAt *time.Time `json:"detected_at,omitempty"`
}

type AssignIncidentRequest struct {
	OwnerID       string     `json:"owner_id"`
	NextAction    string     `json:"next_action,omitempty"`
	NextUpdateDue *time.Time `json:"next_update_due,omitempty"`
	Note          string     `json:"note,omitempty"`
}

type TransitionIncidentRequest struct {
	// Status is acknowledged, in_progress, restored, verified or closed.
	Status string `json:"status" example:"restored"`
	// At is when power was actually restored (restored) or when the
	// confirming observation was made (verified).
	At *time.Time `json:"at,omitempty"`
	// Note is required to verify (what was observed) and to reopen.
	Note string `json:"note,omitempty"`
	// RootCause is required to close.
	RootCause string `json:"root_cause,omitempty"`
}

type IncidentUpdateRequest struct {
	Body string `json:"body"`
	// CustomerVisible shares this update with Airtel; the incident must be published.
	CustomerVisible bool `json:"customer_visible,omitempty"`
	// The plan fields are supervisor-only. A restoration target is a plan,
	// not a restoration.
	NextAction        *string    `json:"next_action,omitempty"`
	NextUpdateDue     *time.Time `json:"next_update_due,omitempty"`
	RestorationTarget *time.Time `json:"restoration_target,omitempty"`
}

// IncidentDTO is the internal view, for Coloplus staff.
type IncidentDTO struct {
	ID                string     `json:"id"`
	Ref               string     `json:"ref" example:"INC-000123"`
	SiteID            string     `json:"site_id"`
	Category          string     `json:"category"`
	Severity          string     `json:"severity"`
	Impact            string     `json:"impact"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"started_at"`
	DetectedAt        time.Time  `json:"detected_at"`
	ReporterID        string     `json:"reporter_id"`
	OwnerID           string     `json:"owner_id,omitempty"`
	NextAction        string     `json:"next_action,omitempty"`
	NextUpdateDue     *time.Time `json:"next_update_due,omitempty"`
	Overdue           bool       `json:"overdue"`
	RestorationTarget *time.Time `json:"restoration_target,omitempty"`
	RestoredAt        *time.Time `json:"restored_at,omitempty"`
	RestoredBy        string     `json:"restored_by,omitempty"`
	VerifiedAt        *time.Time `json:"verified_at,omitempty"`
	VerifiedBy        string     `json:"verified_by,omitempty"`
	ClosedAt          *time.Time `json:"closed_at,omitempty"`
	RootCause         string     `json:"root_cause,omitempty"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type IncidentUpdateDTO struct {
	ID              string     `json:"id"`
	AuthorID        string     `json:"author_id"`
	Kind            string     `json:"kind"`
	FromStatus      string     `json:"from_status,omitempty"`
	ToStatus        string     `json:"to_status,omitempty"`
	Body            string     `json:"body,omitempty"`
	CustomerVisible bool       `json:"customer_visible"`
	ObservedAt      *time.Time `json:"observed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// PublishedIncidentDTO is what Airtel sees: no internal description, owner,
// notes, root cause or author, only the updates written for the customer.
type PublishedIncidentDTO struct {
	ID                string               `json:"id"`
	Ref               string               `json:"ref"`
	SiteID            string               `json:"site_id"`
	Category          string               `json:"category"`
	Severity          string               `json:"severity"`
	Impact            string               `json:"impact"`
	Title             string               `json:"title"`
	Status            string               `json:"status"`
	StartedAt         time.Time            `json:"started_at"`
	RestorationTarget *time.Time           `json:"restoration_target,omitempty"`
	RestoredAt        *time.Time           `json:"restored_at,omitempty"`
	Verified          bool                 `json:"verified"`
	ClosedAt          *time.Time           `json:"closed_at,omitempty"`
	Updates           []PublishedUpdateDTO `json:"updates,omitempty"`
}

type PublishedUpdateDTO struct {
	At   time.Time `json:"at"`
	Body string    `json:"body"`
}

type IncidentDetailResponse struct {
	Incident  *IncidentDTO          `json:"incident,omitempty"`
	Published *PublishedIncidentDTO `json:"published,omitempty"`
	// Updates is the full history for staff; Airtel's updates are inside Published.
	Updates []IncidentUpdateDTO `json:"updates,omitempty"`
}

type IncidentListResponse struct {
	Incidents []IncidentDTO          `json:"incidents,omitempty"`
	Published []PublishedIncidentDTO `json:"published,omitempty"`
	Total     int                    `json:"total"`
	Limit     int                    `json:"limit"`
	Offset    int                    `json:"offset"`
}

type IncidentSummaryResponse struct {
	// Open is every incident not yet closed; Active is those still awaiting restoration.
	Open     int `json:"open"`
	Active   int `json:"active"`
	Critical int `json:"critical_open"`
	Overdue  int `json:"overdue_updates"`
}
