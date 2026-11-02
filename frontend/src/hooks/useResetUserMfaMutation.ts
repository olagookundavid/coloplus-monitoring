import { useMutation, useQueryClient } from '@tanstack/react-query'

import { resetUserMfa } from '../services'
import { MANAGED_USERS_QUERY_KEY } from './useManagedUsers'

export function useResetUserMfaMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => resetUserMfa(id),
    onSuccess: () => {
      // mfa_enabled flips to false on reset -- the panel's "2-step login"
      // row should reflect that without a manual refresh.
      queryClient.invalidateQueries({ queryKey: MANAGED_USERS_QUERY_KEY })
    },
  })
}
