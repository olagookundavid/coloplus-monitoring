import { useQuery } from '@tanstack/react-query'

import { fetchUsersWithSites } from '../services'

export const MANAGED_USERS_QUERY_KEY = ['managed-users']

export function useManagedUsers() {
  return useQuery({
    queryKey: MANAGED_USERS_QUERY_KEY,
    queryFn: () => fetchUsersWithSites(),
  })
}
