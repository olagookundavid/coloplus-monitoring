import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  IncidentCategory,
  IncidentImpact,
  IncidentSeverity,
  IncidentSummary,
  IncidentUpdateEntry,
  IncidentUpdateKind,
  IncidentWorkflowStatus,
  ReporterIncident,
  ReporterIncidentDetail,
} from '../types'

interface IncidentDTO {
  id: string
  ref: string
  site_id: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  description: string
  status: IncidentWorkflowStatus
  started_at: string
  detected_at: string
  owner_id?: string
  next_action?: string
  next_update_due?: string
  overdue: boolean
  restored_at?: string
  verified_at?: string
  closed_at?: string
  root_cause?: string
  published_at?: string
  created_at: string
  updated_at: string
}

interface IncidentSummaryDTO {
  open: number
  active: number
  critical_open: number
  overdue_updates: number
}

interface IncidentUpdateDTO {
  id: string
  author_id: string
  kind: IncidentUpdateKind
  from_status?: IncidentWorkflowStatus
  to_status?: IncidentWorkflowStatus
  body?: string
  customer_visible: boolean
  observed_at?: string
  created_at: string
}

interface IncidentListResponseDTO {
  incidents?: IncidentDTO[]
  total: number
}

interface IncidentDetailResponseDTO {
  incident?: IncidentDTO
  updates?: IncidentUpdateDTO[]
}

export function mapIncident(dto: IncidentDTO): ReporterIncident {
  return {
    id: dto.id,
    ref: dto.ref,
    siteId: dto.site_id,
    category: dto.category,
    severity: dto.severity,
    impact: dto.impact,
    title: dto.title,
    description: dto.description,
    status: dto.status,
    startedAt: dto.started_at,
    detectedAt: dto.detected_at,
    ownerId: dto.owner_id,
    nextAction: dto.next_action,
    nextUpdateDue: dto.next_update_due,
    overdue: dto.overdue,
    restoredAt: dto.restored_at,
    verifiedAt: dto.verified_at,
    closedAt: dto.closed_at,
    rootCause: dto.root_cause,
    publishedAt: dto.published_at,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

function mapUpdate(dto: IncidentUpdateDTO): IncidentUpdateEntry {
  return {
    id: dto.id,
    kind: dto.kind,
    fromStatus: dto.from_status,
    toStatus: dto.to_status,
    body: dto.body ?? '',
    customerVisible: dto.customer_visible,
    observedAt: dto.observed_at,
    createdAt: dto.created_at,
    authorId: dto.author_id,
  }
}

// One call, no status filter -- a reporter's assigned-site incident volume
// is small, so bucketing Open/Awaiting review/Closed/All happens client-side
// from this single list rather than one round trip per tab.
export async function fetchReporterIncidents(): Promise<ReporterIncident[]> {
  try {
    const { data } = await getApi().get<IncidentListResponseDTO>('/incidents', {
      params: { limit: 200 },
    })
    return (data.incidents ?? []).map(mapIncident)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load incidents.')
  }
}

export async function fetchReporterIncidentDetail(id: string): Promise<ReporterIncidentDetail> {
  try {
    const { data } = await getApi().get<IncidentDetailResponseDTO>(`/incidents/${id}`)
    if (!data.incident) {
      throw new Error('This incident is not available.')
    }
    return {
      incident: mapIncident(data.incident),
      updates: (data.updates ?? []).map(mapUpdate),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this incident.')
  }
}

export interface CreateIncidentInput {
  siteId: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  description: string
  startedAt: string
}

export async function createIncident(input: CreateIncidentInput): Promise<ReporterIncident> {
  try {
    const { data } = await getApi().post<IncidentDTO>(`/sites/${input.siteId}/incidents`, {
      category: input.category,
      severity: input.severity,
      impact: input.impact,
      title: input.title,
      description: input.description,
      started_at: input.startedAt,
    })
    return mapIncident(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not log this incident.')
  }
}

export async function addIncidentUpdate(id: string, body: string): Promise<IncidentUpdateEntry> {
  try {
    const { data } = await getApi().post<IncidentUpdateDTO>(`/incidents/${id}/updates`, { body })
    return mapUpdate(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not add this update.')
  }
}

export async function restoreIncident(
  id: string,
  at: string,
  note?: string
): Promise<ReporterIncident> {
  try {
    const { data } = await getApi().post<IncidentDTO>(`/incidents/${id}/transition`, {
      status: 'restored',
      at,
      note,
    })
    return mapIncident(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not mark this incident restored.')
  }
}

// Below: supervisor-only actions (incidents.manage). The reporter actions
// above (createIncident, addIncidentUpdate, restoreIncident) stay as they
// are -- these are additive, not a replacement.

export interface AssignIncidentInput {
  ownerId: string
  nextAction?: string
  note?: string
}

export async function assignIncident(
  id: string,
  input: AssignIncidentInput
): Promise<ReporterIncident> {
  try {
    const { data } = await getApi().post<IncidentDTO>(`/incidents/${id}/assign`, {
      owner_id: input.ownerId,
      next_action: input.nextAction,
      note: input.note,
    })
    return mapIncident(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not assign this incident.')
  }
}

export interface TransitionIncidentInput {
  status: IncidentWorkflowStatus
  at?: string
  note?: string
  rootCause?: string
}

// Acknowledge/verify/close -- restore stays on the dedicated restoreIncident
// above since that's the one transition a reporter (not just a supervisor)
// can make.
export async function transitionIncident(
  id: string,
  input: TransitionIncidentInput
): Promise<ReporterIncident> {
  try {
    const { data } = await getApi().post<IncidentDTO>(`/incidents/${id}/transition`, {
      status: input.status,
      at: input.at,
      note: input.note,
      root_cause: input.rootCause,
    })
    return mapIncident(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update this incident.')
  }
}

export async function publishIncident(id: string): Promise<ReporterIncident> {
  try {
    const { data } = await getApi().post<IncidentDTO>(`/incidents/${id}/publish`)
    return mapIncident(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not publish this incident.')
  }
}

export async function fetchIncidentSummary(): Promise<IncidentSummary> {
  try {
    const { data } = await getApi().get<IncidentSummaryDTO>('/incidents/summary')
    return {
      open: data.open,
      active: data.active,
      critical: data.critical_open,
      overdue: data.overdue_updates,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the incident summary.')
  }
}
