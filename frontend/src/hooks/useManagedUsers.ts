import { useQuery } from '@tanstack/react-query'

import { fetchUsers } from '../services'

export const MANAGED_USERS_QUERY_KEY = ['managed-users']

// `enabled` defaults to true for every existing admin-only caller; pages an
// auditor can also reach (lacking users.manage) pass `can('users.manage')`
// so this doesn't fire a doomed 403 request just to resolve display names.
export function useManagedUsers(enabled = true) {
  return useQuery({
    queryKey: MANAGED_USERS_QUERY_KEY,
    queryFn: fetchUsers,
    enabled,
  })
}
