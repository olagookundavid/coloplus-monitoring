import { useQuery } from '@tanstack/react-query'

import { fetchAdminSites } from '../services'

export function useAdminSites() {
  return useQuery({
    queryKey: ['admin-sites'],
    queryFn: () => fetchAdminSites(),
  })
}
