import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { AuditEvent as AuditLogEvent } from '../types'

export interface AuditEvent {
  id: number
  occurredAt: string
  actorId?: string
  actorName?: string
  action: string
  entityType: string
  entityId?: string
}

interface AuditEventDTO {
  id: number
  occurred_at: string
  actor_id?: string
  actor_name?: string
  action: string
  entity_type: string
  entity_id?: string
}

function mapAuditEvent(dto: AuditEventDTO): AuditEvent {
  return {
    id: dto.id,
    occurredAt: dto.occurred_at,
    actorId: dto.actor_id,
    actorName: dto.actor_name,
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

interface AuditLogEventDTO {
  id: number
  occurred_at: string
  actor_id?: string
  actor_name?: string
  action: string
  entity_type: string
  entity_id?: string
  entity_name?: string
  metadata: Record<string, unknown>
}

function mapAuditLogEvent(dto: AuditLogEventDTO): AuditLogEvent {
  return {
    id: dto.id,
    occurredAt: dto.occurred_at,
    actorId: dto.actor_id ?? '',
    actorName: dto.actor_name,
    action: dto.action,
    entityType: dto.entity_type,
    entityId: dto.entity_id ?? '',
    entityName: dto.entity_name,
    metadata: dto.metadata ?? {},
  }
}

export async function fetchAuditEvents(params: {
  actorId?: string
  from?: string
  limit?: number
  offset?: number
}): Promise<{ events: AuditLogEvent[]; total: number }> {
  try {
    const { data } = await getApi().get<{
      events?: AuditLogEventDTO[]
      total: number
    }>('/audit', {
      params: {
        actor_id: params.actorId,
        from: params.from,
        limit: params.limit ?? 200,
        offset: params.offset,
      },
    })
    return { events: (data.events ?? []).map(mapAuditLogEvent), total: data.total }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the audit log.')
  }
}
