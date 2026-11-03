import { useQuery } from '@tanstack/react-query'

import { fetchAdminSites } from '../services'

export const ADMIN_SITES_QUERY_KEY = ['admin-sites']

export function useAdminSites() {
  return useQuery({
    queryKey: ADMIN_SITES_QUERY_KEY,
    queryFn: () => fetchAdminSites(),
  })
}
