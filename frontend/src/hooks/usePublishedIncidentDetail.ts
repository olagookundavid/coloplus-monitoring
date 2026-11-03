import { useQuery } from '@tanstack/react-query'

import { fetchPublishedIncidentDetail } from '../services'

export const PUBLISHED_INCIDENT_QUERY_KEY = ['published-incident']

export function usePublishedIncidentDetail(id: string | null | undefined) {
  return useQuery({
    queryKey: [...PUBLISHED_INCIDENT_QUERY_KEY, id],
    queryFn: () => fetchPublishedIncidentDetail(id as string),
    enabled: Boolean(id),
  })
}
