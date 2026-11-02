import { useQuery } from '@tanstack/react-query'

import { fetchIncidentCorrections } from '../services'

export const INCIDENT_CORRECTIONS_QUERY_KEY = ['incident-corrections']

export function useIncidentCorrections() {
  return useQuery({
    queryKey: INCIDENT_CORRECTIONS_QUERY_KEY,
    queryFn: fetchIncidentCorrections,
  })
}
