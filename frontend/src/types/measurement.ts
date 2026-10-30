export interface PortfolioSummary {
  energyConsumedKwh: number
  dieselRemainingLitres: number
  sitesBelowDieselThreshold: number
  powerAvailabilityPercent: number
  sitesReportedAvailableCount: number
  radioAvailabilityPercent: number
  sitesWithRadioDataCount: number
  totalSitesInView: number
}

export type TrendMetric = 'availability' | 'energy'

export interface PerformanceTrendPoint {
  /** ISO 8601. */
  timestamp: string
  powerAvailabilityPercent: number | null
  radioAvailabilityPercent: number | null
  energyKwh: number | null
}
