import { useQuery } from '@tanstack/react-query'

import { fetchAlerts } from '../services'

export const ALERTS_QUERY_KEY = ['alerts']

// Powers both the Alert Centre's badge and its dropdown panel
// (NotificationsMenu.tsx) off the same TanStack Query cache -- when both are
// mounted with matching params, the second subscriber reuses the
// already-fetched data instead of firing a second request.
export function useAlerts(params: { unread?: boolean; limit?: number } = {}) {
  const limit = params.limit ?? 20
  return useQuery({
    queryKey: [...ALERTS_QUERY_KEY, { unread: params.unread, limit }],
    queryFn: () => fetchAlerts({ unread: params.unread, limit }),
  })
}
