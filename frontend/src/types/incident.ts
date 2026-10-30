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
