// Mirrors dto.TicketDTO (backend/internal/dto/tickets.go) -- real-backend-
// shaped, replacing the old mock Ticket/localStorage pipeline.

// Mirrors domain.TicketCategory.
export type TicketCategory =
  | 'power_outage'
  | 'low_fuel'
  | 'generator_fault'
  | 'equipment_fault'
  | 'access_restriction'
  | 'other'

// Mirrors domain.TicketPriority -- what the requester asked for, not an
// incident severity; Coloplus cannot change it.
export type TicketPriority = 'low' | 'medium' | 'high' | 'critical'

// Mirrors domain.TicketStatus.
export type TicketStatus = 'open' | 'in_progress' | 'resolved'

export type TicketEventKind = 'comment' | 'status' | 'link'

export interface TicketEvent {
  id: string
  /** airtel or coloplus. */
  party: 'airtel' | 'coloplus'
  kind: TicketEventKind
  fromStatus?: TicketStatus
  toStatus?: TicketStatus
  body: string
  /** Only ever sent to Coloplus staff; stays undefined for Airtel. */
  internal?: boolean
  authorId?: string
  authorName?: string
  createdAt: string
}

export interface Ticket {
  id: string
  /** Server-formatted, e.g. "TKT-000123". */
  ref: string
  siteId: string
  siteName?: string
  category: TicketCategory
  priority: TicketPriority
  subject: string
  description: string
  contactEmail: string
  status: TicketStatus
  resolution?: string
  resolvedAt?: string
  reopenCount: number
  /** Absent until the linked incident has been published to Airtel. */
  incidentId?: string
  incidentRef?: string
  createdAt: string
  updatedAt: string
  /** Coloplus's own view; left out for Airtel. */
  raisedBy?: string
  raisedByName?: string
  assigneeId?: string
  assigneeName?: string
  resolvedBy?: string
  resolvedByName?: string
}

export interface TicketDetail {
  ticket: Ticket
  events: TicketEvent[]
}

export interface TicketSummary {
  open: number
  inProgress: number
  resolved: number
  criticalOpen: number
}

export interface RaiseTicketInput {
  siteId: string
  category: TicketCategory
  priority: TicketPriority
  subject: string
  description: string
  contactEmail?: string
}
