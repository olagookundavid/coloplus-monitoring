import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { Alert, AlertListResult } from '../types'

interface AlertDTO {
  id: string
  kind: string
  title: string
  body: string
  created_at: string
  acknowledged_at?: string
}

interface AlertListResponseDTO {
  alerts?: AlertDTO[]
  total: number
  unread: number
  limit: number
  offset: number
}

function mapAlert(dto: AlertDTO): Alert {
  return {
    id: dto.id,
    kind: dto.kind,
    title: dto.title,
    body: dto.body,
    createdAt: dto.created_at,
    acknowledgedAt: dto.acknowledged_at,
  }
}

export interface AlertFilter {
  unread?: boolean
  limit?: number
  offset?: number
}

export async function fetchAlerts(filter: AlertFilter = {}): Promise<AlertListResult> {
  try {
    const { data } = await getApi().get<AlertListResponseDTO>('/alerts', {
      params: {
        unread: filter.unread ? 'true' : undefined,
        limit: filter.limit ?? 20,
        offset: filter.offset,
      },
    })
    return {
      alerts: (data.alerts ?? []).map(mapAlert),
      total: data.total,
      unread: data.unread,
      limit: data.limit,
      offset: data.offset,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load your alerts.')
  }
}

export async function acknowledgeAlert(id: string): Promise<void> {
  try {
    await getApi().post(`/alerts/${id}/acknowledge`, {})
  } catch (error) {
    throw mapAxiosError(error, 'Could not acknowledge this alert.')
  }
}

export async function acknowledgeAllAlerts(): Promise<{ acknowledged: number }> {
  try {
    const { data } = await getApi().post<{ acknowledged: number }>('/alerts/acknowledge-all', {})
    return data
  } catch (error) {
    throw mapAxiosError(error, 'Could not acknowledge your alerts.')
  }
}
