import { useQuery } from '@tanstack/react-query'

import { fetchReporterIncidents } from '../services'

export const REPORTER_INCIDENTS_QUERY_KEY = ['reporter-incidents']

export function useReporterIncidents() {
  return useQuery({
    queryKey: REPORTER_INCIDENTS_QUERY_KEY,
    queryFn: fetchReporterIncidents,
  })
}
