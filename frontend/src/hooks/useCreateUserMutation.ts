import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createUser } from '../services'
import type { CreateUserRequest } from '../services'
import { MANAGED_USERS_QUERY_KEY } from './useManagedUsers'

export function useCreateUserMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateUserRequest) => createUser(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MANAGED_USERS_QUERY_KEY })
    },
  })
}
