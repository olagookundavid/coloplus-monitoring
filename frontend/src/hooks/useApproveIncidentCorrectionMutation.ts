import { useMutation, useQueryClient } from '@tanstack/react-query'

import { approveIncidentCorrection } from '../services'
import { INCIDENT_CORRECTIONS_QUERY_KEY } from './useIncidentCorrections'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'

export function useApproveIncidentCorrectionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => approveIncidentCorrection(id),
    onSuccess: (correction) => {
      queryClient.invalidateQueries({ queryKey: INCIDENT_CORRECTIONS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      // Approval mutates the incident's own started_at/restored_at.
      queryClient.invalidateQueries({
        queryKey: [...REPORTER_INCIDENT_QUERY_KEY, correction.incidentId],
      })
    },
  })
}
