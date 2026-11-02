import { useQuery } from '@tanstack/react-query'

import { fetchAuditEvents } from '../services'

export function useAuditEvents(params: { actorId?: string; from?: string; limit?: number }) {
  const limit = params.limit ?? 200
  return useQuery({
    queryKey: ['audit-events', { ...params, limit }],
    queryFn: () => fetchAuditEvents({ ...params, limit }),
  })
}
