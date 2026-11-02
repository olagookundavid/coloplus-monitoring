import { useMutation, useQueryClient } from '@tanstack/react-query'

import { addIncidentUpdate } from '../services'
import { REPORTER_INCIDENT_QUERY_KEY } from './useReporterIncidentDetail'

export function useAddIncidentUpdateMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) => addIncidentUpdate(id, body),
    onSuccess: (_update, { id }) => {
      queryClient.invalidateQueries({ queryKey: [...REPORTER_INCIDENT_QUERY_KEY, id] })
    },
  })
}
