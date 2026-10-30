import { getApi } from './api'
import { mapAxiosError } from './auth'

export interface AuditEvent {
  id: number
  occurredAt: string
  actorId?: string
  action: string
  entityType: string
  entityId?: string
}

interface AuditEventDTO {
  id: number
  occurred_at: string
  actor_id?: string
  action: string
  entity_type: string
  entity_id?: string
}

function mapAuditEvent(dto: AuditEventDTO): AuditEvent {
  return {
    id: dto.id,
    occurredAt: dto.occurred_at,
    actorId: dto.actor_id,
    action: dto.action,
    entityType: dto.entity_type,
    entityId: dto.entity_id,
  }
}

export async function fetchRecentAuditEvents(limit = 10): Promise<AuditEvent[]> {
  try {
    const { data } = await getApi().get<{ events: AuditEventDTO[] }>('/audit', {
      params: { limit },
    })
    return data.events.map(mapAuditEvent)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load audit activity.')
  }
}
