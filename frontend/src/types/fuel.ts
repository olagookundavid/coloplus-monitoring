// A derived view model for the Fuel health card -- computed client-side
// from PortfolioSite[].fuel (site.ts), not its own fetch. Litres/hours
// remaining replace the old mock's percent-of-capacity figure: the real
// FuelBriefDTO doesn't carry tank capacity, only the estimate itself.
export interface FuelHealthSite {
  siteId: string
  siteName: string
  litres: number | null
  hoursRemaining: number | null
  low: boolean
  empty: boolean
}

export interface FuelHealthSummary {
  totalLitresRemaining: number
  /** Sites flagged low or empty, not every site. */
  sites: FuelHealthSite[]
  sitesNeedingReplenishment: number
}
