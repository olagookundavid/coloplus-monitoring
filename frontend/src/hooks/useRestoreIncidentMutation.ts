import { useMutation, useQueryClient } from '@tanstack/react-query'

import { restoreIncident } from '../services'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'

export function useRestoreIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, at, note }: { id: string; at: string; note?: string }) =>
      restoreIncident(id, at, note),
    onSuccess: (_incident, { id }) => {
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id] })
    },
  })
}
