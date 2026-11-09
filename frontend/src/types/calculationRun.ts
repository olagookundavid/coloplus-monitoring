export interface CalculationRunRow {
  siteId: string
  customerSiteCode: string
  siteName: string
  eligibleMinutes: number
  servedMinutes: number
  outageMinutes: number
  unknownMinutes: number
  assumedMinutes: number
  daysInPeriod: number
  daysReported: number
  reportedAvailabilityPercent: number | null
  confirmedServedSharePercent: number | null
  coveragePercent: number | null
  generatorHours: number
  incidentsStarted: number
  reportsFingerprint: string
}

export interface CalculationRun {
  id: string
  kind: string
  /** "YYYY-MM-DD" */
  periodFrom: string
  periodTo: string
  region: string
  label: string
  formula: string
  formulaVersion: string
  siteCount: number
  createdBy: string
  createdByName?: string
  createdAt: string
  /** Present on GET /calculation-runs/:id, absent on the list and on
      create -- every site's frozen figures. */
  rows?: CalculationRunRow[]
}

export interface CalculationRunListResult {
  runs: CalculationRun[]
  total: number
  limit: number
  offset: number
}

export interface RunChange {
  siteId: string
  customerSiteCode: string
  siteName: string
  /** Absent when the site didn't exist/wasn't in the run back then. */
  then?: CalculationRunRow
  /** Absent when the site no longer appears (deleted, or out of region). */
  now?: CalculationRunRow
  reportsChanged: boolean
}

export interface RunComparison {
  run: CalculationRun
  generatedAt: string
  formulaChanged: boolean
  currentFormulaVersion: string
  unchanged: number
  changes: RunChange[]
}

export interface CreateCalculationRunInput {
  month: string
  region?: string
  label?: string
}
