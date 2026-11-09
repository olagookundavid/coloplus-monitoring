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
	// PowerStatus, when set, is the reporter's simplified day-level entry:
	// "served" (power on all day) or "outage" (had an outage). It builds the
	// day's single interval and Intervals is ignored.
	PowerStatus string `json:"power_status,omitempty" example:"served"`
	// GeneratorHours is required only when PowerStatus is "outage", 0-24.
	GeneratorHours *float64 `json:"generator_hours,omitempty" example:"3.5"`
	// OutageReason says why the site was unavailable. A site is available
	// unless someone says otherwise, so a report that records an outage cannot
	// be submitted without it.
	OutageReason string `json:"outage_reason,omitempty" example:"Grid down; generator failed to start"`
}

type SaveReportRequest struct {
	// RowVersion is the version this edit was based on. A stale one is
	// rejected with a conflict.
	RowVersion  int           `json:"row_version"`
	Notes       string        `json:"notes"`
	Intervals   []IntervalDTO `json:"intervals"`
	PowerStatus string        `json:"power_status,omitempty" example:"served"`
	// GeneratorHours is required only when PowerStatus is "outage", 0-24.
	GeneratorHours *float64 `json:"generator_hours,omitempty" example:"3.5"`
	// OutageReason: see CreateReportRequest.
	OutageReason string `json:"outage_reason,omitempty"`
}

// SubmitRequest is optional. A report submitted after its site would already
// show "Awaiting update" must say why it is late.
type SubmitRequest struct {
	LateReason string `json:"late_reason,omitempty" example:"Site had no network until Tuesday"`
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
	SiteName         string        `json:"site_name,omitempty" names:"SiteID:site"`
	ReportDate       string        `json:"report_date"`
	Revision         int           `json:"revision"`
	Status           string        `json:"status"`
	AuthorID         string        `json:"author_id"`
	AuthorName       string        `json:"author_name,omitempty" names:"AuthorID:user"`
	Notes            string        `json:"notes"`
	Intervals        []IntervalDTO `json:"intervals,omitempty"`
	SubmittedAt      *time.Time    `json:"submitted_at,omitempty"`
	ReviewedAt       *time.Time    `json:"reviewed_at,omitempty"`
	ReviewerID       string        `json:"reviewer_id,omitempty"`
	ReviewerName     string        `json:"reviewer_name,omitempty" names:"ReviewerID:user"`
	ReviewComment    string        `json:"review_comment,omitempty"`
	SupersedesID     string        `json:"supersedes_id,omitempty"`
	SupersedesLabel  string        `json:"supersedes_label,omitempty" names:"SupersedesID:daily_report"`
	CorrectionReason string        `json:"correction_reason,omitempty"`
	RowVersion       int           `json:"row_version"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	GeneratorHours   *float64      `json:"generator_hours,omitempty"`
	// LateReason is why the report was submitted late; absent when it was on time.
	LateReason string `json:"late_reason,omitempty"`
	// OutageReason is why the site was unavailable that day.
	OutageReason string `json:"outage_reason,omitempty"`
}

// PublishedReportDTO is what the customer sees: approved facts only. Internal notes,
// author and reviewer details and review commentary are left out.
type PublishedReportDTO struct {
	ID             string        `json:"id"`
	SiteID         string        `json:"site_id"`
	SiteName       string        `json:"site_name,omitempty" names:"SiteID:site"`
	ReportDate     string        `json:"report_date"`
	Revision       int           `json:"revision"`
	Status         string        `json:"status"`
	Intervals      []IntervalDTO `json:"intervals,omitempty"`
	ApprovedAt     *time.Time    `json:"approved_at,omitempty"`
	GeneratorHours *float64      `json:"generator_hours,omitempty"`
}

// ReportStatusDTO is a lightweight status-only view of a report, for a
// table cell rather than a detail page.
type ReportStatusDTO struct {
	ReportID      string `json:"report_id"`
	Status        string `json:"status"`
	RowVersion    int    `json:"row_version"`
	ReviewComment string `json:"review_comment,omitempty"`
}

type SummaryDTO struct {
	EligibleMinutes int `json:"eligible_minutes"`
	ServedMinutes   int `json:"served_minutes"`
	OutageMinutes   int `json:"outage_minutes"`
	UnknownMinutes  int `json:"unknown_minutes"`
	// The percentages are null when not applicable or unavailable, never 0.
	CoveragePct             *float64 `json:"coverage_pct"`
	ReportedAvailabilityPct *float64 `json:"reported_availability_pct"`
	// AssumedAvailableMinutes is time counted as available because nobody
	// reported otherwise (a site is available by default). It is part of
	// reported_availability_pct but never of confirmed_served_share_pct, so the
	// confirmed and the assumed parts can always be told apart.
	AssumedAvailableMinutes int `json:"assumed_available_minutes"`
	// ExcludedMaintenanceMinutes is approved planned maintenance, already taken
	// out of eligible_minutes. AssumedOutageMinutes is an outage assumed to
	// continue from the day before until someone reported; it counts against
	// reported availability but is not confirmed and not part of coverage.
	ExcludedMaintenanceMinutes int `json:"excluded_maintenance_minutes"`
	AssumedOutageMinutes       int `json:"assumed_outage_minutes"`
	// SourceMinutes is how long each power source was reported serving the site
	// (grid, generator, battery, ...), each counted once. Sources can overlap,
	// so they need not add up to served_minutes. Absent when none was reported.
	SourceMinutes           map[string]int `json:"source_minutes,omitempty"`
	ConfirmedServedSharePct *float64       `json:"confirmed_served_share_pct"`
	Provisional             bool           `json:"provisional"`
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

// ReportDetailResponse holds either Report (internal) or Published (customer).
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
	// Date is the last day of the period, kept for clients that ask for one day.
	Date string `json:"date"`
	// From and To are the first and last reporting day pooled into the figures.
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
	// Region is the filter applied, empty for every region; Regions lists the
	// regions in the caller's scope for the filter to offer.
	Region              string    `json:"region"`
	Regions             []string  `json:"regions"`
	Timezone            string    `json:"timezone"`
	GeneratedAt         time.Time `json:"generated_at"`
	ActiveSites         int       `json:"active_sites"`
	PendingSites        int       `json:"pending_acceptance_sites"`
	SuspendedSites      int       `json:"suspended_sites"`
	SitesReported       int       `json:"sites_reported"`
	SitesNotReported    int       `json:"sites_not_reported"`
	SitesAwaitingUpdate int       `json:"sites_awaiting_update"`
	SitesOutageReported int       `json:"sites_outage_reported"`
	// SiteDaysWithoutReport counts site-days in the period with no approved report.
	SiteDaysWithoutReport int `json:"site_days_without_report"`
	// DataQuality is the indicator for the active sites together.
	DataQuality DataQualityDTO `json:"data_quality"`
	// SLA judges the portfolio's figure against the target; absent when none is set.
	SLA *SLADTO `json:"sla,omitempty"`
	// OutagesThisWeek counts distinct site/day pairs in the trailing 7 days
	// (inclusive of the requested date) whose current report has an outage.
	// An approximation, not a full weekly aggregation.
	OutagesThisWeek int          `json:"outages_this_week"`
	Portfolio       SummaryDTO   `json:"portfolio"`
	Sites           []SiteRowDTO `json:"sites"`
}

type SiteRowDTO struct {
	SiteID           string `json:"site_id"`
	CustomerSiteCode string `json:"customer_site_code"`
	ColoplusCode     string `json:"coloplus_code"`
	Name             string `json:"name"`
	SiteType         string `json:"site_type"`
	State            string `json:"state"`
	Status           string `json:"status"`
	// Condition is available, outage_reported or awaiting_update.
	Condition string     `json:"condition" example:"available"`
	AsOf      *time.Time `json:"as_of,omitempty"`
	Region    string     `json:"region"`
	// ResponsibleTeam is the team accountable for the site.
	ResponsibleTeam string `json:"responsible_team"`
	// CurrentSource is what the site was last observed running on; absent when
	// the report is stale, the last interval was an outage or none was observed.
	CurrentSource string `json:"current_source,omitempty" example:"grid"`
	// Summary pools the period; absent when no day of it has an approved report.
	Summary *SummaryDTO `json:"summary,omitempty"`
	// DaysReported of the response's Days have an approved report.
	DaysReported int `json:"days_reported"`
	// DaysWithoutReport are the days of the period nobody reported. For a site
	// that is available by default they are counted as available (assumed).
	DaysWithoutReport int `json:"days_without_report"`
	// DataQuality says how much of the figure rests on reports.
	DataQuality DataQualityDTO `json:"data_quality"`
	// SLA judges the figure against the agreed target; absent when no target is set
	// or the figure does not apply.
	SLA *SLADTO `json:"sla,omitempty"`
	// Fuel is the site's estimated diesel; absent when you may not see fuel, or
	// the site has no agreed rate or nothing recorded.
	Fuel *FuelBriefDTO `json:"fuel,omitempty"`

	// TodayStatus/YesterdayStatus are the report that best represents that
	// day's live status (in-progress if there is one, else the approved
	// one); nil when nothing has ever been filed. For the reporter's own
	// progress view, not the approved-only Summary/Condition above.
	TodayStatus     *ReportStatusDTO `json:"today_status,omitempty"`
	YesterdayStatus *ReportStatusDTO `json:"yesterday_status,omitempty"`
}

// TrendResponse is the portfolio day by day.
type TrendResponse struct {
	// Grain is day, week or month: the width of each point.
	Grain       string       `json:"grain" example:"week"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	Region      string       `json:"region"`
	Timezone    string       `json:"timezone"`
	Formula     string       `json:"formula" example:"daily-availability"`
	Version     string       `json:"version" example:"1"`
	GeneratedAt time.Time    `json:"generated_at"`
	Points      []TrendPoint `json:"points"`
}

// TrendPoint is the active portfolio on one reporting day. Its percentages are
// null when nothing was reported, so a chart shows a gap, not a line.
type TrendPoint struct {
	// Date is the first day of the bucket: the day, the Monday of the week, or
	// the first of the month. From and To are the days of it inside the
	// requested period, so a partial first or last bucket is plain.
	Date string `json:"date"`
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
	// SitesReported had an approved report on at least one day of the bucket.
	ActiveSites   int `json:"active_sites"`
	SitesReported int `json:"sites_reported"`
	// SiteDays are active sites times days; SiteDaysReported how many of them
	// have an approved report.
	SiteDays         int        `json:"site_days"`
	SiteDaysReported int        `json:"site_days_reported"`
	Summary          SummaryDTO `json:"summary"`
}

// DayBarDTO is one day of the reporter's "my sites" trailing-days strip.
type DayBarDTO struct {
	Date string `json:"date" example:"2026-10-31"`
	// Color is green (reported, no outage), amber (reported, had an outage)
	// or red (no approved report that day -- a visible gap).
	Color string `json:"color" example:"green"`
}

// MySiteRowDTO is one card on the reporter's "my sites" grid.
type MySiteRowDTO struct {
	SiteID             string `json:"site_id"`
	CustomerSiteCode   string `json:"customer_site_code"`
	ColoplusCode       string `json:"coloplus_code"`
	Name               string `json:"name"`
	SiteType           string `json:"site_type"`
	Address            string `json:"address"`
	State              string `json:"state"`
	GeneratorCapacity  string `json:"generator_capacity"`
	AccessContactName  string `json:"access_contact_name"`
	AccessContactPhone string `json:"access_contact_phone"`
	SupervisorName     string `json:"supervisor_name"`
	// TodayStatus is the report that best represents today's live status
	// (in-progress if there is one, else approved); nil when nothing has
	// ever been filed today.
	TodayStatus *ReportStatusDTO `json:"today_status,omitempty"`
	// AvailabilityPct30d is the pooled reported availability over the
	// trailing 30 days. Null when the site has no approved report in the
	// window -- never 0.
	AvailabilityPct30d *float64 `json:"availability_pct_30d"`
	OpenIncidents      int      `json:"open_incidents"`
	// Bars is the trailing 7 days, oldest first (last entry is today).
	Bars []DayBarDTO `json:"bars"`
	// Fuel is the site's estimated diesel; absent when you may not see fuel, or
	// the site has no agreed rate or nothing recorded.
	Fuel *FuelBriefDTO `json:"fuel,omitempty"`
}

type MySitesResponse struct {
	Date        string         `json:"date"`
	Timezone    string         `json:"timezone"`
	GeneratedAt time.Time      `json:"generated_at"`
	Sites       []MySiteRowDTO `json:"sites"`
}

// DataQualityDTO says how much of a figure rests on reports. The level is a
// plain summary of the numbers beside it; its thresholds are proposed defaults
// (good: at least 95% of days reported and at most a fifth of reports late;
// fair: at least 80% of days; otherwise poor).
type DataQualityDTO struct {
	// Level is good, fair or poor.
	Level string `json:"level" example:"good"`
	Days  int    `json:"days"`
	// ReportedDays have an approved report; Completeness is that as a percent.
	ReportedDays int     `json:"reported_days"`
	Completeness float64 `json:"completeness_pct"`
	// LateReports were submitted late (and say why); Corrections are approved
	// corrections of earlier reports.
	LateReports int `json:"late_reports"`
	Corrections int `json:"corrections"`
	// UnconfirmedSharePct is the share of eligible time no report confirmed.
	UnconfirmedSharePct float64 `json:"unconfirmed_share_pct"`
}

// SLADTO judges an availability figure against the agreed target. Status is
// met, at_risk or missed. The figure judged is the reported availability, which
// may include time that was only assumed; confirmed_meets_target says whether
// the reported time alone, assuming nothing, reaches the target.
type SLADTO struct {
	TargetPercent        float64 `json:"target_percent" example:"99.5"`
	Status               string  `json:"status" example:"met"`
	ConfirmedMeetsTarget bool    `json:"confirmed_meets_target"`
}

// FuelBriefDTO is a site's estimated diesel on one line. An estimate, never a
// meter reading; GET /sites/{id}/fuel says how it was reached.
type FuelBriefDTO struct {
	Basis          string   `json:"basis" example:"estimate"`
	Litres         *float64 `json:"litres"`
	HoursRemaining *float64 `json:"hours_remaining"`
	// Low is true at or under the low-fuel threshold; Empty when the estimate ran out.
	Low   bool `json:"low"`
	Empty bool `json:"empty"`
	// Provisional: days since the last reading have no approved hours, so the
	// level is the most the tank could hold.
	Provisional bool `json:"provisional"`
}
