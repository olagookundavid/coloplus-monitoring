import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { Notification, NotificationListResult, NotificationStatus } from '../types'

interface NotificationDTO {
  id: string
  kind: string
  recipient: string
  subject: string
  status: NotificationStatus
  attempts: number
  last_error?: string
  created_at: string
  sent_at?: string
}

interface NotificationListResponseDTO {
  notifications?: NotificationDTO[]
  total: number
  limit: number
  offset: number
}

function mapNotification(dto: NotificationDTO): Notification {
  return {
    id: dto.id,
    kind: dto.kind,
    recipient: dto.recipient,
    subject: dto.subject,
    status: dto.status,
    attempts: dto.attempts,
    lastError: dto.last_error,
    createdAt: dto.created_at,
    sentAt: dto.sent_at,
  }
}

export interface NotificationFilter {
  status?: NotificationStatus
  limit?: number
  offset?: number
}

export async function fetchNotifications(
  filter: NotificationFilter = {}
): Promise<NotificationListResult> {
  try {
    const { data } = await getApi().get<NotificationListResponseDTO>('/notifications', {
      params: {
        status: filter.status,
        limit: filter.limit ?? 200,
        offset: filter.offset,
      },
    })
    return {
      notifications: (data.notifications ?? []).map(mapNotification),
      total: data.total,
      limit: data.limit,
      offset: data.offset,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the email log.')
  }
}

export async function retryNotification(id: string): Promise<void> {
  try {
    await getApi().post(`/notifications/${id}/retry`, {})
  } catch (error) {
    throw mapAxiosError(error, 'Could not retry this email.')
  }
}
