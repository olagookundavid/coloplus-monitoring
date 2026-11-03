import { useQuery } from '@tanstack/react-query'

import { fetchDisputes } from '../services'
import type { DisputeFilter } from '../services'

export const DISPUTES_QUERY_KEY = ['disputes']

export function useDisputes(filter: DisputeFilter = {}) {
  return useQuery({
    queryKey: [...DISPUTES_QUERY_KEY, filter],
    queryFn: () => fetchDisputes(filter),
  })
}
