import { useMutation, useQueryClient } from '@tanstack/react-query'

import { requestMaintenance } from '../services'
import { MAINTENANCE_QUERY_KEY } from './useMaintenanceWindows'

export interface RequestMaintenanceMutationInput {
  siteId: string
  startsAt: string
  endsAt: string
  reason: string
}

export function useRequestMaintenanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ siteId, startsAt, endsAt, reason }: RequestMaintenanceMutationInput) =>
      requestMaintenance(siteId, { startsAt, endsAt, reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MAINTENANCE_QUERY_KEY })
    },
  })
}
