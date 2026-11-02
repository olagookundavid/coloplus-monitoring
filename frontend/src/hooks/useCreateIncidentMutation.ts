import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createIncident, type CreateIncidentInput } from '../services'
import { MY_SITES_QUERY_KEY } from './useMySites'
import { REPORTER_INCIDENTS_QUERY_KEY } from './useReporterIncidents'

export function useCreateIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateIncidentInput) => createIncident(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: REPORTER_INCIDENTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_SITES_QUERY_KEY })
    },
  })
}
