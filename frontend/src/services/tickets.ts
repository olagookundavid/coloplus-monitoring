import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  RaiseTicketInput,
  Ticket,
  TicketCategory,
  TicketEvent,
  TicketEventKind,
  TicketPriority,
  TicketStatus,
  TicketSummary,
} from '../types'

interface TicketDTO {
  id: string
  ref: string
  site_id: string
  site_name?: string
  category: TicketCategory
  priority: TicketPriority
  subject: string
  description: string
  contact_email: string
  status: TicketStatus
  resolution?: string
  resolved_at?: string
  reopen_count: number
  incident_id?: string
  incident_ref?: string
  created_at: string
  updated_at: string
  raised_by?: string
  raised_by_name?: string
  assignee_id?: string
  assignee_name?: string
  resolved_by?: string
  resolved_by_name?: string
}

interface TicketEventDTO {
  id: string
  party: 'airtel' | 'coloplus'
  kind: TicketEventKind
  from_status?: TicketStatus
  to_status?: TicketStatus
  body?: string
  internal?: boolean
  author_id?: string
  author_name?: string
  created_at: string
}

interface TicketListResponseDTO {
  tickets?: TicketDTO[]
  total: number
}

interface TicketSummaryDTO {
  open: number
  in_progress: number
  resolved: number
  critical_open: number
}

function mapTicket(dto: TicketDTO): Ticket {
  return {
    id: dto.id,
    ref: dto.ref,
    siteId: dto.site_id,
    siteName: dto.site_name,
    category: dto.category,
    priority: dto.priority,
    subject: dto.subject,
    description: dto.description,
    contactEmail: dto.contact_email,
    status: dto.status,
    resolution: dto.resolution,
    resolvedAt: dto.resolved_at,
    reopenCount: dto.reopen_count,
    incidentId: dto.incident_id,
    incidentRef: dto.incident_ref,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
    raisedBy: dto.raised_by,
    raisedByName: dto.raised_by_name,
    assigneeId: dto.assignee_id,
    assigneeName: dto.assignee_name,
    resolvedBy: dto.resolved_by,
    resolvedByName: dto.resolved_by_name,
  }
}

function mapEvent(dto: TicketEventDTO): TicketEvent {
  return {
    id: dto.id,
    party: dto.party,
    kind: dto.kind,
    fromStatus: dto.from_status,
    toStatus: dto.to_status,
    body: dto.body ?? '',
    internal: dto.internal,
    authorId: dto.author_id,
    authorName: dto.author_name,
    createdAt: dto.created_at,
  }
}

export interface TicketFilter {
  siteId?: string
  status?: TicketStatus
  openOnly?: boolean
  incidentId?: string
}

export async function fetchTickets(filter: TicketFilter = {}): Promise<Ticket[]> {
  try {
    const { data } = await getApi().get<TicketListResponseDTO>('/tickets', {
      params: {
        site_id: filter.siteId,
        status: filter.status,
        open: filter.openOnly ? 'true' : undefined,
        incident_id: filter.incidentId,
        limit: 200,
      },
    })
    return (data.tickets ?? []).map(mapTicket)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load support tickets.')
  }
}

export async function fetchTicketDetail(
  id: string
): Promise<{ ticket: Ticket; events: TicketEvent[] }> {
  try {
    const { data } = await getApi().get<{ ticket: TicketDTO; events?: TicketEventDTO[] }>(
      `/tickets/${id}`
    )
    return { ticket: mapTicket(data.ticket), events: (data.events ?? []).map(mapEvent) }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this ticket.')
  }
}

export async function fetchTicketSummary(): Promise<TicketSummary> {
  try {
    const { data } = await getApi().get<TicketSummaryDTO>('/tickets/summary')
    return {
      open: data.open,
      inProgress: data.in_progress,
      resolved: data.resolved,
      criticalOpen: data.critical_open,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the ticket summary.')
  }
}

export async function raiseTicket(input: RaiseTicketInput): Promise<Ticket> {
  try {
    const clientKey =
      typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : undefined
    const { data } = await getApi().post<TicketDTO>(`/sites/${input.siteId}/tickets`, {
      category: input.category,
      priority: input.priority,
      subject: input.subject,
      description: input.description,
      contact_email: input.contactEmail,
      client_key: clientKey,
    })
    return mapTicket(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not create this ticket.')
  }
}

export async function commentTicket(
  id: string,
  body: string,
  internal = false
): Promise<TicketEvent> {
  try {
    const { data } = await getApi().post<TicketEventDTO>(`/tickets/${id}/comments`, {
      body,
      internal,
    })
    return mapEvent(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not add this comment.')
  }
}

// Coloplus's general transition (start work, resolve, take a resolved
// ticket back into work). Airtel's reopen keeps its own reopenTicket()
// below, which always sends status "open" -- this one never does.
export interface TransitionTicketInput {
  status: 'in_progress' | 'resolved'
  note?: string
}

export async function transitionTicket(id: string, input: TransitionTicketInput): Promise<Ticket> {
  try {
    const { data } = await getApi().post<TicketDTO>(`/tickets/${id}/transition`, {
      status: input.status,
      note: input.note,
    })
    return mapTicket(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update this ticket.')
  }
}

export async function linkTicketIncident(ticketId: string, incidentId: string): Promise<Ticket> {
  try {
    const { data } = await getApi().post<TicketDTO>(`/tickets/${ticketId}/incident`, {
      incident_id: incidentId,
    })
    return mapTicket(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not link this ticket to that incident.')
  }
}

export async function reopenTicket(id: string, note: string): Promise<Ticket> {
  try {
    const { data } = await getApi().post<TicketDTO>(`/tickets/${id}/transition`, {
      status: 'open',
      note,
    })
    return mapTicket(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not reopen this ticket.')
  }
}
