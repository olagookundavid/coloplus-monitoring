import { useMutation, useQueryClient } from '@tanstack/react-query'

import { requestIncidentCorrection } from '../services'
import { INCIDENT_CORRECTIONS_QUERY_KEY } from './useIncidentCorrections'
import type { IncidentCorrectionField } from '../types'

export interface RequestIncidentCorrectionInput {
  incidentId: string
  field: IncidentCorrectionField
  correctedValue: string
  reason: string
}

export function useRequestIncidentCorrectionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ incidentId, field, correctedValue, reason }: RequestIncidentCorrectionInput) =>
      requestIncidentCorrection(incidentId, field, correctedValue, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: INCIDENT_CORRECTIONS_QUERY_KEY })
    },
  })
}
