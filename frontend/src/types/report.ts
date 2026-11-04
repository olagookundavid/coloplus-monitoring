import type { FuelBrief } from './site'

export type ReportStatus = 'draft' | 'submitted' | 'approved' | 'rejected' | 'superseded'

// The reporter's simplified day-level toggle, and the matching state of the
// single interval the backend builds from it (backend/internal/domain
// .DayLevelInterval). Mirrors backend's domain.IntervalState.
export type PowerStatus = 'served' | 'outage'

// Mirrors dto.IntervalDTO. Only used to derive a report's PowerStatus for
// display/editing -- the reporter dashboard never edits intervals directly.
export interface PowerInterval {
  start: string
  end: string
  state: PowerStatus
  source?: string
  basis?: 'observed' | 'declared'
}

// Mirrors backend/internal/dto.ReportDTO. Intervals are optional -- the
// admin Approvals screen never lists or edits them, only the summary
// derived from them; the reporter dashboard loads them to derive the
// power-status toggle's initial value.
export interface DailyReport {
  id: string
  siteId: string
  reportDate: string
  revision: number
  status: ReportStatus
  authorId: string
  notes: string
  intervals?: PowerInterval[]
  generatorHours?: number | null
  outageReason?: string
  lateReason?: string
  submittedAt?: string
  reviewedAt?: string
  reviewerId?: string
  reviewComment?: string
  supersedesId?: string
  correctionReason?: string
  rowVersion: number
  createdAt: string
  updatedAt: string
}

// Mirrors dto.ReportStatusDTO -- a status-only view for a table cell.
export interface ReportStatusSummary {
  reportId: string
  status: ReportStatus
  rowVersion: number
  reviewComment?: string
}

// One row of the reporter's "My sites today" table -- mirrors the
// reporter-relevant fields of dto.SiteRowDTO.
export interface SiteReportRow {
  siteId: string
  airtelCode: string
  coloplusCode: string
  name: string
  siteType: string
  state: string
  todayStatus?: ReportStatusSummary
  yesterdayStatus?: ReportStatusSummary
  /** Absent when the caller may not see fuel, or the site has no agreed rate/entry. */
  fuel?: FuelBrief
}

// Mirrors dto.OverviewResponse's reporter-relevant fields.
export interface DashboardOverview {
  date: string
  timezone: string
  generatedAt: string
  outagesThisWeek: number
  sites: SiteReportRow[]
}

// Mirrors dto.DayBarDTO -- one day of the "my sites" trailing-days strip.
// green = reported, no outage; amber = reported, had an outage; red = no
// approved report that day -- a visible gap, never fabricated.
export type DayBarColor = 'green' | 'amber' | 'red'

export interface DayBar {
  date: string
  color: DayBarColor
}

// One card on the reporter's "My sites" grid -- mirrors dto.MySiteRowDTO.
// Extends SiteReportRow so a MySiteRow can be passed straight into
// DailyReportPanel's `site` prop with no adapter.
export interface MySiteRow extends SiteReportRow {
  address: string
  generatorCapacity: string
  accessContactName: string
  accessContactPhone: string
  supervisorName: string
  // Pooled reported availability over the trailing 30 days. Null when the
  // site has no approved report in the window -- never 0.
  availabilityPct30d: number | null
  openIncidents: number
  // Trailing 7 days, oldest first.
  bars: DayBar[]
}

// Mirrors dto.MySitesResponse.
export interface MySitesResponse {
  date: string
  timezone: string
  generatedAt: string
  sites: MySiteRow[]
}

// Mirrors dto.SummaryDTO. Percentages are null when not applicable or
// unavailable, never 0 -- frontend/CLAUDE.md §11.
export interface ReportSummary {
  eligibleMinutes: number
  servedMinutes: number
  outageMinutes: number
  unknownMinutes: number
  coveragePct: number | null
  reportedAvailabilityPct: number | null
  confirmedServedSharePct: number | null
  provisional: boolean
}

// Mirrors dto.CalculationDTO -- powers a "how calculated" explanation.
export interface ReportCalculation {
  formula: string
  version: string
  basis: string
  inputReportId: string
  inputRevision: number
  windowStart: string
  windowEnd: string
  timezone: string
  computedAt: string
  explanation: string
  warnings: string[]
}

export interface ReportDetail {
  report: DailyReport
  summary: ReportSummary
  calculation: ReportCalculation
  openDisputes: number
}
