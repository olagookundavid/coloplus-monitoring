import { useMutation, useQueryClient } from '@tanstack/react-query'

import { retryNotification } from '../services'
import { NOTIFICATIONS_QUERY_KEY } from './useNotifications'

export function useRetryNotificationMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => retryNotification(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_QUERY_KEY })
    },
  })
}
