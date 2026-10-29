package dto

import "time"

type IntervalDTO struct {
	ID    string    `json:"id,omitempty"`
	Start time.Time `json:"start" example:"2026-12-01T06:00:00+01:00"`
	End   time.Time `json:"end" example:"2026-12-01T18:30:00+01:00"`
	// State is served or outage.
	State string `json:"state" example:"served"`
	// Source is required when served and must be empty for an outage.
	Source string `json:"source,omitempty" example:"grid"`
	// Basis is observed (default) or declared. Only observed counts.
	Basis string `json:"basis,omitempty" example:"observed"`
}

type CreateReportRequest struct {
	ReportDate string        `json:"report_date" example:"2026-12-01"`
	Notes      string        `json:"notes"`
	Intervals  []IntervalDTO `json:"intervals"`
}

type SaveReportRequest struct {
	// RowVersion is the version this edit was based on. A stale one is
	// rejected with a conflict.
	RowVersion int           `json:"row_version"`
	Notes      string        `json:"notes"`
	Intervals  []IntervalDTO `json:"intervals"`
}

type ReviewRequest struct {
	Comment string `json:"comment"`
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

type CorrectionRequest struct {
	Reason string `json:"reason"`
}

// ReportDTO is the internal view, for Coloplus staff.
type ReportDTO struct {
	ID               string        `json:"id"`
	SiteID           string        `json:"site_id"`
	ReportDate       string        `json:"report_date"`
	Revision         int           `json:"revision"`
	Status           string        `json:"status"`
	AuthorID         string        `json:"author_id"`
	Notes            string        `json:"notes"`
	Intervals        []IntervalDTO `json:"intervals,omitempty"`
	SubmittedAt      *time.Time    `json:"submitted_at,omitempty"`
	ReviewedAt       *time.Time    `json:"reviewed_at,omitempty"`
	ReviewerID       string        `json:"reviewer_id,omitempty"`
	ReviewComment    string        `json:"review_comment,omitempty"`
	SupersedesID     string        `json:"supersedes_id,omitempty"`
	CorrectionReason string        `json:"correction_reason,omitempty"`
	RowVersion       int           `json:"row_version"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// PublishedReportDTO is what Airtel sees: approved facts only. Internal notes,
// author and reviewer details and review commentary are left out.
type PublishedReportDTO struct {
	ID         string        `json:"id"`
	SiteID     string        `json:"site_id"`
	ReportDate string        `json:"report_date"`
	Revision   int           `json:"revision"`
	Status     string        `json:"status"`
	Intervals  []IntervalDTO `json:"intervals,omitempty"`
	ApprovedAt *time.Time    `json:"approved_at,omitempty"`
}

type SummaryDTO struct {
	EligibleMinutes int `json:"eligible_minutes"`
	ServedMinutes   int `json:"served_minutes"`
	OutageMinutes   int `json:"outage_minutes"`
	UnknownMinutes  int `json:"unknown_minutes"`
	// The percentages are null when not applicable or unavailable, never 0.
	CoveragePct             *float64 `json:"coverage_pct"`
	ReportedAvailabilityPct *float64 `json:"reported_availability_pct"`
	ConfirmedServedSharePct *float64 `json:"confirmed_served_share_pct"`
	Provisional             bool     `json:"provisional"`
}

// CalculationDTO powers the "How calculated" panel.
type CalculationDTO struct {
	Formula     string    `json:"formula" example:"daily-availability"`
	Version     string    `json:"version" example:"1"`
	Basis       string    `json:"basis" example:"calculation from reported inputs"`
	InputReport string    `json:"input_report_id"`
	InputRev    int       `json:"input_revision"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	Timezone    string    `json:"timezone" example:"Africa/Lagos"`
	ComputedAt  time.Time `json:"computed_at"`
	Explanation string    `json:"explanation"`
	Warnings    []string  `json:"warnings"`
}

// ReportDetailResponse holds either Report (internal) or Published (Airtel).
type ReportDetailResponse struct {
	Report      *ReportDTO          `json:"report,omitempty"`
	Published   *PublishedReportDTO `json:"published,omitempty"`
	Summary     SummaryDTO          `json:"summary"`
	Calculation CalculationDTO      `json:"calculation"`
	// OpenDisputes counts the disputes about this report that have not closed.
	// A disputed figure keeps showing it.
	OpenDisputes int `json:"open_disputes"`
}

type ReportListResponse struct {
	Reports   []ReportDTO          `json:"reports,omitempty"`
	Published []PublishedReportDTO `json:"published,omitempty"`
	// OpenDisputes maps a report id to its number of unclosed disputes;
	// reports with none are left out.
	OpenDisputes map[string]int `json:"open_disputes,omitempty"`
	Total        int            `json:"total"`
	Limit        int            `json:"limit"`
	Offset       int            `json:"offset"`
}

type OverviewResponse struct {
	Date                string       `json:"date"`
	Timezone            string       `json:"timezone"`
	GeneratedAt         time.Time    `json:"generated_at"`
	ActiveSites         int          `json:"active_sites"`
	PendingSites        int          `json:"pending_acceptance_sites"`
	SuspendedSites      int          `json:"suspended_sites"`
	SitesReported       int          `json:"sites_reported"`
	SitesNotReported    int          `json:"sites_not_reported"`
	SitesAwaitingUpdate int          `json:"sites_awaiting_update"`
	SitesOutageReported int          `json:"sites_outage_reported"`
	Portfolio           SummaryDTO   `json:"portfolio"`
	Sites               []SiteRowDTO `json:"sites"`
}

type SiteRowDTO struct {
	SiteID       string `json:"site_id"`
	AirtelCode   string `json:"airtel_code"`
	ColoplusCode string `json:"coloplus_code"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Status       string `json:"status"`
	// Condition is available, outage_reported or awaiting_update.
	Condition string      `json:"condition" example:"available"`
	AsOf      *time.Time  `json:"as_of,omitempty"`
	Summary   *SummaryDTO `json:"summary,omitempty"`
}
