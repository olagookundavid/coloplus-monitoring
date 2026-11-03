import { useMutation, useQueryClient } from '@tanstack/react-query'

import { rejectIncidentCorrection } from '../services'
import { INCIDENT_CORRECTIONS_QUERY_KEY } from './useIncidentCorrections'

export interface RejectIncidentCorrectionInput {
  id: string
  note: string
}

export function useRejectIncidentCorrectionMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: RejectIncidentCorrectionInput) => rejectIncidentCorrection(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: INCIDENT_CORRECTIONS_QUERY_KEY })
    },
  })
}
