import { useMutation, useQueryClient } from '@tanstack/react-query'

import { acknowledgeAllAlerts } from '../services'
import { ALERTS_QUERY_KEY } from './useAlerts'

export function useAcknowledgeAllAlertsMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => acknowledgeAllAlerts(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ALERTS_QUERY_KEY })
    },
  })
}
