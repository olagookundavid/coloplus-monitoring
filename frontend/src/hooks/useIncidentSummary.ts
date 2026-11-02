import { useQuery } from '@tanstack/react-query'

import { fetchIncidentSummary } from '../services'

export const INCIDENT_SUMMARY_QUERY_KEY = ['incident-summary']

export function useIncidentSummary() {
  return useQuery({
    queryKey: INCIDENT_SUMMARY_QUERY_KEY,
    queryFn: fetchIncidentSummary,
  })
}
