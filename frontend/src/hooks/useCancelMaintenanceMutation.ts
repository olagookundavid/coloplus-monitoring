import { useMutation, useQueryClient } from '@tanstack/react-query'

import { cancelMaintenance } from '../services'
import { MAINTENANCE_QUERY_KEY } from './useMaintenanceWindows'

export interface CancelMaintenanceInput {
  id: string
  note: string
}

export function useCancelMaintenanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: CancelMaintenanceInput) => cancelMaintenance(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MAINTENANCE_QUERY_KEY })
    },
  })
}
