import { useMutation, useQueryClient } from '@tanstack/react-query'

import { approveMaintenance } from '../services'
import { MAINTENANCE_QUERY_KEY } from './useMaintenanceWindows'

export interface ApproveMaintenanceInput {
  id: string
  note?: string
}

export function useApproveMaintenanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: ApproveMaintenanceInput) => approveMaintenance(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MAINTENANCE_QUERY_KEY })
    },
  })
}
