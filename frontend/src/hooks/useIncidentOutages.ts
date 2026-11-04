import { useQuery } from '@tanstack/react-query'

import { fetchIncidentOutages } from '../services'

export const INCIDENT_OUTAGES_QUERY_KEY = ['incident-outages']

export function useIncidentOutages(incidentId: string | null | undefined) {
  return useQuery({
    queryKey: [...INCIDENT_OUTAGES_QUERY_KEY, incidentId],
    queryFn: () => fetchIncidentOutages(incidentId as string),
    enabled: Boolean(incidentId),
  })
}
