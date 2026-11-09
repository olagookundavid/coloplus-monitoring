// The customer-facing site shape -- mirrors dto.SiteRowDTO from GET
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
  /** Approved planned maintenance, already taken out of eligibleMinutes. */
  excludedMaintenanceMinutes: number
  /** An outage assumed to continue from the day before until someone
      reported; lowers reported availability but isn't confirmed or part of
      coverage. */
  assumedOutageMinutes: number
  /** The confirmed-only share -- never includes assumedAvailableMinutes. */
  confirmedServedSharePct: number | null
  provisional: boolean
  /** How long each power source was reported serving the site (grid,
      generator, ...), each counted once. Sources can overlap, so these need
      not add up to servedMinutes. Absent when none was reported. */
  sourceMinutes?: Record<string, number>
}

// Judges an availability figure against the agreed target. The figure judged
// is reportedAvailabilityPct, which may include assumed time;
// confirmedMeetsTarget says whether the confirmed time alone, assuming
// nothing, also reaches it -- show both, never merge them into one badge.
export interface SLA {
  targetPercent: number
  status: 'met' | 'at_risk' | 'missed'
  confirmedMeetsTarget: boolean
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
  customerSiteCode: string
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
  /** Absent when no target is set or the figure doesn't apply. */
  sla?: SLA
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
  /** Absent when no target is set. */
  sla?: SLA
  outagesThisWeek: number
  portfolio: PeriodSummary
  sites: PortfolioSite[]
}

export interface PortfolioQueryParams {
  region?: string
  date?: string
  from?: string
  to?: string
  /** Keeps only site rows at this data-quality level; the counts and the
      portfolio figures still cover every site. */
  quality?: 'good' | 'fair' | 'poor'
}

// Mirrors dto.TrendPoint from GET /dashboard/trend. summary's percentages are
// null on a day nobody reported -- a chart gap, never a line at 0 or 100.
export interface PortfolioTrendPoint {
  /** The first day of the bucket: the day, the Monday of the week, or the
      first of the month. */
  date: string
  /** The days of the bucket inside the requested period -- a partial first
      or last bucket is plain. */
  from: string
  to: string
  days: number
  activeSites: number
  sitesReported: number
  /** Active sites times days. */
  siteDays: number
  /** How many site-days of siteDays have an approved report. */
  siteDaysReported: number
  summary: PeriodSummary
}

// Mirrors dto.TrendResponse. formula/version aren't carried -- nothing on
// the frontend surfaces a "how calculated" panel for this pooled figure.
export interface PortfolioTrend {
  grain: 'day' | 'week' | 'month'
  from: string
  to: string
  region: string
  timezone: string
  generatedAt: string
  points: PortfolioTrendPoint[]
}

// GET /dashboard/trend requires both bounds, unlike /dashboard/overview's
// optional date|from|to -- a dedicated params type, not PortfolioQueryParams.
export interface PortfolioTrendQueryParams {
  region?: string
  from: string
  to: string
  /** day (default), week (Monday to Sunday) or month. */
  grain?: 'day' | 'week' | 'month'
}
