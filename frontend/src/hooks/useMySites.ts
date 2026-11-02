import { useQuery } from '@tanstack/react-query'

import { fetchMySites } from '../services'

export const MY_SITES_QUERY_KEY = ['my-sites']

export function useMySites() {
  return useQuery({
    queryKey: MY_SITES_QUERY_KEY,
    queryFn: fetchMySites,
  })
}
