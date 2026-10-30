export type TicketStatus = 'Open' | 'In progress' | 'Resolved'
export type TicketPriority = 'Low' | 'Medium' | 'High' | 'Critical'
export type TicketCategory =
  | 'Power outage'
  | 'Low fuel / diesel'
  | 'Generator fault'
  | 'Equipment fault'
  | 'Access restriction'
  | 'Other'

export interface Ticket {
  id: string
  subject: string
  siteCode: string
  category: TicketCategory
  priority: TicketPriority
  status: TicketStatus
  contactEmail: string
  description: string
  attachmentNames: string[]
  /** ISO 8601. */
  createdAt: string
}

export interface CreateTicketRequest {
  siteCode: string
  category: TicketCategory
  subject: string
  priority: TicketPriority
  contactEmail: string
  description: string
  attachmentNames: string[]
}
