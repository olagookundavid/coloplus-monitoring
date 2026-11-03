import { useQuery } from '@tanstack/react-query'

import { fetchNotifications } from '../services'
import type { NotificationStatus } from '../types'

export const NOTIFICATIONS_QUERY_KEY = ['notifications']

export function useNotifications(params: { status?: NotificationStatus; limit?: number } = {}) {
  const limit = params.limit ?? 200
  return useQuery({
    queryKey: [...NOTIFICATIONS_QUERY_KEY, { status: params.status, limit }],
    queryFn: () => fetchNotifications({ status: params.status, limit }),
  })
}
