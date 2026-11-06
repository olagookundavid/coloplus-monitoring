import type { DailyReport } from './report'

// Mirrors domain.CorrectionField.
export type IncidentCorrectionField = 'started_at' | 'restored_at'

// Mirrors domain.CorrectionStatus.
export type IncidentCorrectionStatus = 'requested' | 'approved' | 'rejected'

// A reporter's request to correct a single already-recorded incident field --
// mirrors dto.IncidentCorrectionDTO. There is no backend equivalent for a
// daily-report field: that case reuses the existing report-correction
// (supersedes_id) mechanism instead, see CorrectionListItem below.
export interface IncidentCorrection {
  id: string
  /** Server-formatted, e.g. "COR-000058". */
  ref: string
  incidentId: string
  incidentLabel?: string
  siteId: string
  siteName?: string
  field: IncidentCorrectionField
  originalValue: string
  correctedValue: string
  reason: string
  status: IncidentCorrectionStatus
  requestedBy: string
  requestedByName?: string
  reviewedBy?: string
  reviewedByName?: string
  reviewedAt?: string
  reviewNote?: string
  createdAt: string
  updatedAt: string
}

// The three tabs the Corrections page shows. A rejected incident correction
// surfaces under needsAction too, with a "Request again" action, rather than
// vanishing -- there is no fourth "rejected" tab.
export type CorrectionTab = 'needsAction' | 'awaitingReview' | 'approved'

// One row on the Corrections page. Three different things can land here:
// a report a supervisor returned (fix the same report, no new revision), a
// report correction already in flight (a revision with supersedesId set --
// the existing StartCorrection mechanism, reused, not rebuilt), or a new
// incident-field correction. The detail pane switches on `kind` and reads
// straight from the real object, same precedent as Approvals.tsx's DetailCard.
export type CorrectionListItem =
  | { id: string; kind: 'returnedReport'; tab: CorrectionTab; siteId: string; report: DailyReport }
  | {
      id: string
      kind: 'reportCorrection'
      tab: CorrectionTab
      siteId: string
      report: DailyReport
    }
  | {
      id: string
      kind: 'incidentCorrection'
      tab: CorrectionTab
      siteId: string
      correction: IncidentCorrection
    }
