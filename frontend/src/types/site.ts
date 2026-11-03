// The Airtel-facing site shape -- mirrors dto.SiteRowDTO from GET
// /dashboard/overview (backend/internal/dto/reports.go), the one call that
// already bundles profile, live condition, data quality and fuel per site.
// Real-backend-shaped, unlike the old mock Site this replaced (no more
// invented "radio availability" or "energy consumed" figures -- neither is
// a real tracked metric; only power service and fuel are).

// Mirrors domain.SiteCondition: available, outage_reported or awaiting_update.
export type SiteCondition = 'available' | 'outage_reported' | 'awaiting_update'

export interface DataQuality {
  /** good, fair or poor -- thresholds are proposed defaults, not yet agreed with the customer. */
  level: 'good' | 'fair' | 'poor'
  days: number
  reportedDays: number
  completenessPct: number
  lateReports: number
  corrections: number
  /** Share of eligible time no report confirmed. */
  unconfirmedSharePct: number
}

export interface PeriodSummary {
  eligibleMinutes: number
  servedMinutes: number
  outageMinutes: number
  unknownMinutes: number
  /** Null when not applicable or unavailable -- never 0. */
  coveragePct: number | null
  reportedAvailabilityPct: number | null
  /** Minutes counted as available only because nobody reported otherwise. */
  assumedAvailableMinutes: number
  /** The confirmed-only share -- never includes assumedAvailableMinutes. */
  confirmedServedSharePct: number | null
  provisional: boolean
}

export interface FuelBrief {
  basis: 'estimate'
  litres: number | null
  hoursRemaining: number | null
  low: boolean
  empty: boolean
  provisional: boolean
}

export interface PortfolioSite {
  id: string
  airtelCode: string
  coloplusCode: string
  name: string
  siteType: string
  state: string
  region: string
  status: string
  condition: SiteCondition
  asOf?: string
  responsibleTeam: string
  /** Absent when the report is stale, the last interval was an outage, or none was observed. */
  currentSource?: string
  /** Absent when no day of the period has an approved report -- a gap, never zero. */
  summary?: PeriodSummary
  daysReported: number
  daysWithoutReport: number
  /** Absent if the backend hasn't attached a data-quality indicator to this row. */
  dataQuality?: DataQuality
  /** Absent when the caller may not see fuel, or the site has no agreed rate/entry. */
  fuel?: FuelBrief
}

export interface PortfolioOverview {
  date: string
  from: string
  to: string
  days: number
  region: string
  regions: string[]
  timezone: string
  generatedAt: string
  activeSites: number
  pendingSites: number
  suspendedSites: number
  sitesReported: number
  sitesNotReported: number
  sitesAwaitingUpdate: number
  sitesOutageReported: number
  siteDaysWithoutReport: number
  /** Absent if the backend hasn't attached a data-quality indicator to this response. */
  dataQuality?: DataQuality
  outagesThisWeek: number
  portfolio: PeriodSummary
  sites: PortfolioSite[]
}

export interface PortfolioQueryParams {
  region?: string
  date?: string
  from?: string
  to?: string
}
