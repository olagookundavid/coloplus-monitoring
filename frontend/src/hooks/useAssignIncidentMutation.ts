import { useMutation, useQueryClient } from '@tanstack/react-query'

import { assignIncident } from '../services'
import type { AssignIncidentInput } from '../services'
import { INCIDENT_SUMMARY_QUERY_KEY } from './useIncidentSummary'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'

export function useAssignIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: AssignIncidentInput }) =>
      assignIncident(id, input),
    onSuccess: (_incident, { id }) => {
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: INCIDENT_SUMMARY_QUERY_KEY })
    },
  })
}
