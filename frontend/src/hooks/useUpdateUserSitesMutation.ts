import { useMutation, useQueryClient } from '@tanstack/react-query'

import { updateUserSites } from '../services'
import { MANAGED_USERS_QUERY_KEY } from './useManagedUsers'

export function useUpdateUserSitesMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, siteIds }: { id: string; siteIds: string[] }) =>
      updateUserSites(id, siteIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MANAGED_USERS_QUERY_KEY })
    },
  })
}
