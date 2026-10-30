package dto

import "time"

type RaiseDisputeRequest struct {
	// ReportID is the published report being disputed; leave empty to dispute
	// the site or a period as a whole.
	ReportID string `json:"report_id,omitempty"`
	// IntervalID narrows it to one interval of that report.
	IntervalID string `json:"interval_id,omitempty"`
	// Metric is availability, coverage, outage_minutes, served_minutes or other.
	Metric string `json:"metric,omitempty" example:"availability"`
	// PeriodFrom and PeriodTo are dates (2026-12-01); give both or neither.
	PeriodFrom string `json:"period_from,omitempty" example:"2026-12-01"`
	PeriodTo   string `json:"period_to,omitempty" example:"2026-12-31"`
	// Category is incorrect_outage, missing_outage, incorrect_served,
	// wrong_duration, wrong_metric, missing_report or other.
	Category    string `json:"category" example:"incorrect_outage"`
	Title       string `json:"title" example:"Outage shown on a day power was on"`
	Description string `json:"description"`
}

type TransitionDisputeRequest struct {
	// Status is under_review, awaiting_evidence, agreed_correction, resolved
	// or closed. Coloplus reviews and settles; Airtel withdraws, gives
	// evidence or reopens (under_review), and acknowledges a resolution (closed).
	Status string `json:"status" example:"under_review"`
	// Note explains the step. Needed to ask for evidence, agree a correction,
	// resolve, give evidence, reopen and withdraw.
	Note string `json:"note,omitempty"`
	// Outcome is upheld or not_upheld, required to resolve. Upheld needs the
	// correction to have been agreed first.
	Outcome string `json:"outcome,omitempty" example:"upheld"`
	// ResolutionReportID is the corrected, approved report revision that
	// settles an upheld dispute about a report.
	ResolutionReportID string `json:"resolution_report_id,omitempty"`
}

type DisputeCommentRequest struct {
	Body string `json:"body"`
	// Internal keeps the comment inside Coloplus; Airtel never sees it.
	Internal bool `json:"internal,omitempty"`
}

// DisputeDTO is one dispute. RaisedBy, OwnerID and ResolvedBy are Coloplus's
// own view and are left out for Airtel.
type DisputeDTO struct {
	ID         string `json:"id"`
	Ref        string `json:"ref" example:"DSP-000123"`
	SiteID     string `json:"site_id"`
	ReportID   string `json:"report_id,omitempty"`
	IntervalID string `json:"interval_id,omitempty"`
	Metric     string `json:"metric,omitempty"`
	PeriodFrom string `json:"period_from,omitempty"`
	PeriodTo   string `json:"period_to,omitempty"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	// Description is what Airtel wrote when raising it.
	Description string `json:"description"`
	Status      string `json:"status"`
	// WaitingOn is whose move it is: coloplus or airtel; empty once closed.
	WaitingOn string `json:"waiting_on,omitempty"`
	// Outcome is upheld or not_upheld once Coloplus resolves it, withdrawn if Airtel dropped it.
	Outcome            string `json:"outcome,omitempty"`
	Resolution         string `json:"resolution,omitempty"`
	ResolutionReportID string `json:"resolution_report_id,omitempty"`
	// ResolvedAt is Coloplus's decision; AcknowledgedAt is Airtel's agreement.
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ReopenCount    int        `json:"reopen_count"`
	RaisedBy       string     `json:"raised_by,omitempty"`
	OwnerID        string     `json:"owner_id,omitempty"`
	ResolvedBy     string     `json:"resolved_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DisputeEventDTO struct {
	ID string `json:"id"`
	// Party is airtel or coloplus.
	Party      string `json:"party"`
	Kind       string `json:"kind"`
	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status,omitempty"`
	Body       string `json:"body,omitempty"`
	// Internal and AuthorID are only ever sent to Coloplus staff.
	Internal  bool      `json:"internal,omitempty"`
	AuthorID  string    `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DisputeDetailResponse struct {
	Dispute DisputeDTO        `json:"dispute"`
	Events  []DisputeEventDTO `json:"events"`
}

type DisputeListResponse struct {
	Disputes []DisputeDTO `json:"disputes"`
	Total    int          `json:"total"`
	Limit    int          `json:"limit"`
	Offset   int          `json:"offset"`
}

type DisputeSummaryResponse struct {
	// Open is every dispute not yet closed.
	Open int `json:"open"`
	// WithColoplus are waiting on Coloplus; WithAirtel on Airtel (evidence or acknowledgement).
	WithColoplus int `json:"with_coloplus"`
	WithAirtel   int `json:"with_airtel"`
}
