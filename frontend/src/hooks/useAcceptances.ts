import { useQuery } from '@tanstack/react-query'

import { fetchAcceptances } from '../services'
import type { AcceptanceFilter } from '../services'

export const ACCEPTANCES_QUERY_KEY = ['acceptances']

export function useAcceptances(filter: AcceptanceFilter = {}) {
  return useQuery({
    queryKey: [...ACCEPTANCES_QUERY_KEY, filter],
    queryFn: () => fetchAcceptances(filter),
  })
}
