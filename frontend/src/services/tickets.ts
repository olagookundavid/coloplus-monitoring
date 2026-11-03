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

/**
 * Real backend (replacing the old mock/localStorage pipeline): GET /tickets,
 * POST /sites/:id/tickets, POST /tickets/:id/comments,
 * POST /tickets/:id/transition. Both airtel_viewer and airtel_reviewer hold
 * tickets.raise -- unlike disputes/acceptance, this is not reviewer-gated.
 */

interface TicketDTO {
  id: string
  ref: string
  site_id: string
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
}

interface TicketEventDTO {
  id: string
  party: 'airtel' | 'coloplus'
  kind: TicketEventKind
  from_status?: TicketStatus
  to_status?: TicketStatus
  body?: string
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
    createdAt: dto.created_at,
  }
}

export interface TicketFilter {
  siteId?: string
  status?: TicketStatus
  openOnly?: boolean
}

export async function fetchTickets(filter: TicketFilter = {}): Promise<Ticket[]> {
  try {
    const { data } = await getApi().get<TicketListResponseDTO>('/tickets', {
      params: {
        site_id: filter.siteId,
        status: filter.status,
        open: filter.openOnly ? 'true' : undefined,
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

export async function commentTicket(id: string, body: string): Promise<TicketEvent> {
  try {
    const { data } = await getApi().post<TicketEventDTO>(`/tickets/${id}/comments`, { body })
    return mapEvent(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not add this comment.')
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
