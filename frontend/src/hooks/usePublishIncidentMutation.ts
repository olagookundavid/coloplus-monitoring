import { useMutation, useQueryClient } from '@tanstack/react-query'

import { publishIncident } from '../services'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'

export function usePublishIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => publishIncident(id),
    onSuccess: (_incident, id) => {
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id] })
    },
  })
}
