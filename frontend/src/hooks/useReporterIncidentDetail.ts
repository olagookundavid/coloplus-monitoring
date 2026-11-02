import { useQuery } from '@tanstack/react-query'

import { fetchReporterIncidentDetail } from '../services'

export const REPORTER_INCIDENT_QUERY_KEY = ['reporter-incident']

export function useReporterIncidentDetail(id: string | null | undefined) {
  return useQuery({
    queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id],
    queryFn: () => fetchReporterIncidentDetail(id as string),
    enabled: Boolean(id),
  })
}
