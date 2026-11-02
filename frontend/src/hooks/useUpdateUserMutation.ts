import { useMutation, useQueryClient } from '@tanstack/react-query'

import { updateUser } from '../services'
import type { UpdateUserPatch } from '../services'
import { MANAGED_USERS_QUERY_KEY } from './useManagedUsers'

export function useUpdateUserMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: UpdateUserPatch }) => updateUser(id, patch),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MANAGED_USERS_QUERY_KEY })
    },
  })
}
