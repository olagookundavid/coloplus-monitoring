import { useMutation, useQueryClient } from '@tanstack/react-query'

import { acknowledgeAlert } from '../services'
import { ALERTS_QUERY_KEY } from './useAlerts'

export function useAcknowledgeAlertMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => acknowledgeAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ALERTS_QUERY_KEY })
    },
  })
}
