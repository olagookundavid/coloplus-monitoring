import { useMutation, useQueryClient } from '@tanstack/react-query'

import { transitionIncident } from '../services'
import type { TransitionIncidentInput } from '../services'
import { INCIDENT_SUMMARY_QUERY_KEY } from './useIncidentSummary'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'

export function useTransitionIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TransitionIncidentInput }) =>
      transitionIncident(id, input),
    onSuccess: (_incident, { id }) => {
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: INCIDENT_SUMMARY_QUERY_KEY })
    },
  })
}
