import { useQuery } from '@tanstack/react-query'

import { fetchPublishedIncidents } from '../services'
import type { PublishedIncidentFilter } from '../services'

export const PUBLISHED_INCIDENTS_QUERY_KEY = ['published-incidents']

export function usePublishedIncidents(filter: PublishedIncidentFilter = {}) {
  return useQuery({
    queryKey: [...PUBLISHED_INCIDENTS_QUERY_KEY, filter],
    queryFn: () => fetchPublishedIncidents(filter),
  })
}
