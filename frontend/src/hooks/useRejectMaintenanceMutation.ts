import { useMutation, useQueryClient } from '@tanstack/react-query'

import { rejectMaintenance } from '../services'
import { MAINTENANCE_QUERY_KEY } from './useMaintenanceWindows'

export interface RejectMaintenanceInput {
  id: string
  note: string
}

export function useRejectMaintenanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: RejectMaintenanceInput) => rejectMaintenance(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MAINTENANCE_QUERY_KEY })
    },
  })
}
