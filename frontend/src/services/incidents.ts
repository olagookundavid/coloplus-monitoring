import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  IncidentCategory,
  IncidentImpact,
  IncidentSeverity,
  IncidentWorkflowStatus,
  PublishedIncident,
  PublishedIncidentUpdate,
} from '../types'

interface PublishedUpdateDTO {
  at: string
  body: string
}

interface PublishedIncidentDTO {
  id: string
  ref: string
  site_id: string
  category: IncidentCategory
  severity: IncidentSeverity
  impact: IncidentImpact
  title: string
  status: IncidentWorkflowStatus
  started_at: string
  restoration_target?: string
  restored_at?: string
  verified: boolean
  closed_at?: string
  updates?: PublishedUpdateDTO[]
}

interface IncidentListResponseDTO {
  published?: PublishedIncidentDTO[]
  total: number
}

function mapUpdate(dto: PublishedUpdateDTO): PublishedIncidentUpdate {
  return { at: dto.at, body: dto.body }
}

export function mapPublishedIncident(dto: PublishedIncidentDTO): PublishedIncident {
  return {
    id: dto.id,
    ref: dto.ref,
    siteId: dto.site_id,
    category: dto.category,
    severity: dto.severity,
    impact: dto.impact,
    title: dto.title,
    status: dto.status,
    startedAt: dto.started_at,
    restorationTarget: dto.restoration_target,
    restoredAt: dto.restored_at,
    verified: dto.verified,
    closedAt: dto.closed_at,
    updates: (dto.updates ?? []).map(mapUpdate),
  }
}

export interface PublishedIncidentFilter {
  siteId?: string
  status?: IncidentWorkflowStatus
  openOnly?: boolean
}

export async function fetchPublishedIncidents(
  filter: PublishedIncidentFilter = {}
): Promise<PublishedIncident[]> {
  try {
    const { data } = await getApi().get<IncidentListResponseDTO>('/incidents', {
      params: {
        site_id: filter.siteId,
        status: filter.status,
        open: filter.openOnly ? 'true' : undefined,
        limit: 200,
      },
    })
    return (data.published ?? []).map(mapPublishedIncident)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load incidents.')
  }
}
