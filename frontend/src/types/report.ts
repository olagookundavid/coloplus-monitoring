export type ReportStatus = 'draft' | 'submitted' | 'approved' | 'rejected' | 'superseded'

// Mirrors backend/internal/dto.ReportDTO. Intervals are left out -- the
// admin Approvals screen never lists or edits them, only the summary
// derived from them.
export interface DailyReport {
  id: string
  siteId: string
  reportDate: string
  revision: number
  status: ReportStatus
  authorId: string
  notes: string
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
