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

// Below: the real fuel record/estimate shapes, mirroring
// dto.FuelEntryDTO/FuelVarianceDTO/FuelEstimateDTO/FuelStatusResponse
// (backend/internal/dto/fuel.go) -- used by the fuel-recording UI
// (FuelPanel), a separate concern from the derived summary above.

export type FuelEntryKind = 'delivery' | 'reading'

// A tank reading compared with what the estimate expected just before it.
// The reading wins; the estimate restarts from it.
export interface FuelVariance {
  estimatedBeforeLitres: number
  readingLitres: number
  /** Reading minus estimate; negative means less than expected. */
  differenceLitres: number
  /** The estimate had unreported days, so it was an upper bound -- a
      negative difference is expected, not a reporting problem. */
  provisional: boolean
  /** The gap is too large to be rounding; check the rate, hours or entries. */
  reviewWarning: boolean
}

export interface FuelEntry {
  id: string
  siteId: string
  kind: FuelEntryKind
  litres: number
  occurredAt: string
  /** Coloplus-only; absent for a customer. */
  note?: string
  /** Coloplus-only; absent for a customer. */
  recordedBy?: string
  voided: boolean
  voidReason?: string
  createdAt: string
  /** Present only on a reading. */
  variance?: FuelVariance
}

// Always basis "estimate" -- from entries and the site's agreed litres per
// hour, never a meter reading. litres/hoursRemaining are null when
// unavailable; while unreportedDays is above 0, litres is the most the tank
// could hold, since those days' use is unknown.
export interface FuelEstimate {
  basis: 'estimate'
  available: boolean
  /** Says why there is no estimate, when there is none. */
  reason?: string
  litres: number | null
  hoursRemaining: number | null
  consumptionLPH: number
  anchorKind?: string
  anchorAt?: string
  assumedEmptyStart: boolean
  hoursRun: number
  consumedLitres: number
  unreportedDays: number
  provisional: boolean
  estimatedEmpty: boolean
  /** The estimate exceeds the tank, so an entry is probably wrong. */
  overCapacity: boolean
  tankCapacityLitres: number | null
  asOf: string
  formula: string
  formulaVersion: string
  explanation: string
  /** Run time at or under which the tank counts as low. */
  lowFuelHours: number
  low: boolean
}

export interface FuelStatus {
  siteId: string
  estimate: FuelEstimate
  /** Newest first, capped server-side at 500. */
  entries: FuelEntry[]
}

export interface RecordFuelInput {
  siteId: string
  kind: FuelEntryKind
  litres: number
  /** ISO timestamp; must not be in the future. */
  occurredAt: string
  note?: string
}
