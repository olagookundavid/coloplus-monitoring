export interface FuelHealthSite {
  siteId: string
  siteName: string
  percent: number
}

export interface FuelHealthSummary {
  totalLitresRemaining: number
  /**
   * A curated subset shown on the Fuel health card, not every site. The real
   * selection rule (lowest-N? below-threshold?) is a product question for
   * later, not decided here.
   */
  sites: FuelHealthSite[]
  sitesNeedingReplenishment: number
}
