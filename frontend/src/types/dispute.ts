// Mirrors dto.DisputeDTO (backend/internal/dto/disputes.go). RaisedBy/
// OwnerID/ResolvedBy are Coloplus's own view and left out for Airtel
// (backend/internal/transport/http/mappers.go:297-309) -- always optional
// here, never assumed present.

// Mirrors domain.DisputeCategory.
export type DisputeCategory =
  | 'incorrect_outage'
  | 'missing_outage'
  | 'incorrect_served'
  | 'wrong_duration'
  | 'wrong_metric'
  | 'missing_report'
  | 'other'

// Mirrors domain.DisputeMetric.
export type DisputeMetric = 'availability' | 'coverage' | 'outage_minutes' | 'served_minutes' | 'other'

// Mirrors domain.DisputeStatus.
export type DisputeStatus =
  | 'open'
  | 'under_review'
  | 'awaiting_evidence'
  | 'agreed_correction'
  | 'resolved'
  | 'closed'

export type DisputeParty = 'airtel' | 'coloplus'
export type DisputeOutcome = 'upheld' | 'not_upheld'

export type DisputeEventKind = 'comment' | 'status'

export interface DisputeEvent {
  id: string
  party: DisputeParty
  kind: DisputeEventKind
  fromStatus?: DisputeStatus
  toStatus?: DisputeStatus
  body: string
  createdAt: string
}

export interface Dispute {
  id: string
  /** Server-formatted, e.g. "DSP-000123". */
  ref: string
  siteId: string
  reportId?: string
  intervalId?: string
  metric?: DisputeMetric
  periodFrom?: string
  periodTo?: string
  category: DisputeCategory
  title: string
  description: string
  status: DisputeStatus
  /** Whose move it is -- empty once closed. */
  waitingOn?: DisputeParty
  outcome?: DisputeOutcome
  resolution?: string
  resolutionReportId?: string
  /** Coloplus's decision. */
  resolvedAt?: string
  /** Airtel's agreement -- a resolved dispute isn't closed until this is set. */
  acknowledgedAt?: string
  reopenCount: number
  createdAt: string
  updatedAt: string
}

export interface DisputeDetail {
  dispute: Dispute
  events: DisputeEvent[]
}

export interface DisputeSummary {
  open: number
  withColoplus: number
  withAirtel: number
}

export interface RaiseDisputeInput {
  siteId: string
  reportId?: string
  intervalId?: string
  metric?: DisputeMetric
  periodFrom?: string
  periodTo?: string
  category: DisputeCategory
  title: string
  description: string
}
