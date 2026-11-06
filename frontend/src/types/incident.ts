// Mirrors domain.IncidentStatus. "assigned" is reached only via the assign
// endpoint, not a direct transition.
export type IncidentWorkflowStatus =
  'reported' | 'acknowledged' | 'assigned' | 'in_progress' | 'restored' | 'verified' | 'closed'

// Mirrors domain.Severity.
export type IncidentSeverity = 'minor' | 'major' | 'critical'

// Mirrors domain.IncidentCategory.
export type IncidentCategory =
  | 'grid_loss'
  | 'building_supply_loss'
  | 'generator_failure'
  | 'low_fuel'
  | 'battery_failure'
  | 'rectifier_fault'
  | 'electrical_fault'
  | 'access_restriction'
  | 'other'

// Mirrors domain.Impact.
export type IncidentImpact = 'outage' | 'no_outage'

// Mirrors domain.UpdateKind.
export type IncidentUpdateKind = 'note' | 'status' | 'assignment' | 'plan' | 'customer'

// One row of an incident's timeline -- mirrors dto.IncidentUpdateDTO.
export interface IncidentUpdateEntry {
  id: string
  kind: IncidentUpdateKind
  fromStatus?: IncidentWorkflowStatus
  toStatus?: IncidentWorkflowStatus
  body: string
  customerVisible: boolean
  observedAt?: string
  createdAt: string
  authorId: string
  authorName?: string
}

// One incident on the reporter's Incidents page -- mirrors dto.IncidentDTO's
// reporter-relevant fields. The supervisor Incidents page reuses this same
// shape (the fields below it, e.g. nextAction/overdue/publishedAt, are the
// ones only a supervisor view needs) since GET /incidents is one generic,
// actor-scoped endpoint, not a reporter-specific one.
export interface ReporterIncident {
  id: string
  /** Server-formatted, e.g. "INC-000123". */
  ref: string
  siteId: string
  siteName?: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  description: string
  status: IncidentWorkflowStatus
  startedAt: string
  detectedAt: string
  reporterName?: string
  ownerId?: string
  ownerName?: string
  nextAction?: string
  nextUpdateDue?: string
  overdue: boolean
  restoredAt?: string
  restoredByName?: string
  verifiedAt?: string
  verifiedByName?: string
  closedAt?: string
  rootCause?: string
  publishedAt?: string
  createdAt: string
  updatedAt: string
}

export interface ReporterIncidentDetail {
  incident: ReporterIncident
  updates: IncidentUpdateEntry[]
}

// Mirrors dto.IncidentSummaryResponse (GET /incidents/summary).
export interface IncidentSummary {
  open: number
  active: number
  critical: number
  overdue: number
}

// What Airtel sees -- mirrors dto.PublishedIncidentDTO. No description,
// owner, notes, root cause or author; neither airtel_viewer nor
// airtel_reviewer holds incidents.read, so GET /incidents always returns
// this shape for them, never the internal ReporterIncident above.
export interface PublishedIncidentUpdate {
  at: string
  body: string
}

export interface PublishedIncident {
  id: string
  ref: string
  siteId: string
  siteName?: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  status: IncidentWorkflowStatus
  startedAt: string
  restorationTarget?: string
  restoredAt?: string
  verified: boolean
  closedAt?: string
  updates: PublishedIncidentUpdate[]
}
