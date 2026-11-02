package dto

import "time"

type RaiseTicketRequest struct {
	// Category is power_outage, low_fuel, generator_fault, equipment_fault,
	// access_restriction or other.
	Category string `json:"category" example:"low_fuel"`
	// Priority is the priority you ask for: low, medium, high or critical.
	Priority string `json:"priority" example:"high"`
	Subject  string `json:"subject" example:"Diesel replenishment request"`
	// Description says what happened.
	Description string `json:"description"`
	// ContactEmail is where Coloplus can reach the requester; it defaults to
	// your own address. Coloplus replies by email to your account, not to this.
	ContactEmail string `json:"contact_email,omitempty"`
	// ClientKey is optional. Send the same value when retrying a request that
	// may have gone through, and you get the same ticket back, not a second one.
	ClientKey string `json:"client_key,omitempty"`
}

type TransitionTicketRequest struct {
	// Status is in_progress or resolved for Coloplus, open to reopen a
	// resolved ticket for Airtel.
	Status string `json:"status" example:"in_progress"`
	// Note explains the step. Needed to resolve, to reopen, and to take a
	// resolved ticket back into work.
	Note string `json:"note,omitempty"`
}

type TicketCommentRequest struct {
	Body string `json:"body"`
	// Internal keeps the comment inside Coloplus; Airtel never sees it.
	Internal bool `json:"internal,omitempty"`
}

type LinkIncidentRequest struct {
	IncidentID string `json:"incident_id"`
}

// TicketDTO is one ticket. RaisedBy, AssigneeID and ResolvedBy are Coloplus's
// own view and are left out for Airtel, and so is a linked incident until it
// has been published to them.
type TicketDTO struct {
	ID           string `json:"id"`
	Ref          string `json:"ref" example:"TKT-000123"`
	SiteID       string `json:"site_id"`
	Category     string `json:"category"`
	Priority     string `json:"priority"`
	Subject      string `json:"subject"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	// Status is open, in_progress or resolved.
	Status string `json:"status"`
	// Resolution is how Coloplus resolved it.
	Resolution  string     `json:"resolution,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ReopenCount int        `json:"reopen_count"`
	IncidentID  string     `json:"incident_id,omitempty"`
	IncidentRef string     `json:"incident_ref,omitempty"`
	RaisedBy    string     `json:"raised_by,omitempty"`
	AssigneeID  string     `json:"assignee_id,omitempty"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TicketEventDTO struct {
	ID string `json:"id"`
	// Party is airtel or coloplus.
	Party string `json:"party"`
	// Kind is comment, status or link (staff only).
	Kind       string `json:"kind"`
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status,omitempty"`
	Body       string `json:"body,omitempty"`
	// Internal and AuthorID are only ever sent to Coloplus staff.
	Internal  bool      `json:"internal,omitempty"`
	AuthorID  string    `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type TicketDetailResponse struct {
	Ticket TicketDTO        `json:"ticket"`
	Events []TicketEventDTO `json:"events"`
}

type TicketListResponse struct {
	Tickets []TicketDTO `json:"tickets"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
}

type TicketSummaryResponse struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Resolved   int `json:"resolved"`
	// CriticalOpen are unresolved tickets whose requester asked for critical priority.
	CriticalOpen int `json:"critical_open"`
}
