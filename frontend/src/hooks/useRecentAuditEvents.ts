import { useQuery } from '@tanstack/react-query'

import { fetchRecentAuditEvents } from '../services'

export function useRecentAuditEvents(limit = 10) {
  return useQuery({
    queryKey: ['recent-audit-events', limit],
    queryFn: () => fetchRecentAuditEvents(limit),
  })
}
