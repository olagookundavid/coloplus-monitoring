export type IncidentStatus = 'Open' | 'Resolved'

export interface Incident {
  id: string
  siteCode: string
  title: string
  description: string
  /** The second line in the mockup ("Service remains available. Refill
      required.") — the customer-facing impact statement, distinct from the
      technical description above it. */
  impactNote: string
  status: IncidentStatus
  /** ISO 8601. */
  occurredAt: string
  owner: string
  linkedTicketId?: string
}

// Below: the real reporter-facing incident workflow (backend/internal/domain
// /incident.go), distinctly named from the mock Incident/IncidentStatus
// above (still used by the customer-portal SiteDetailPanel's Fault history
// tab) so the two shapes never collide.

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
}

// One incident on the reporter's Incidents page -- mirrors dto.IncidentDTO's
// reporter-relevant fields.
export interface ReporterIncident {
  id: string
  /** Server-formatted, e.g. "INC-000123". */
  ref: string
  siteId: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  description: string
  status: IncidentWorkflowStatus
  startedAt: string
  detectedAt: string
  ownerId?: string
  restoredAt?: string
  verifiedAt?: string
  closedAt?: string
  rootCause?: string
  createdAt: string
  updatedAt: string
}

export interface ReporterIncidentDetail {
  incident: ReporterIncident
  updates: IncidentUpdateEntry[]
}
