import { useQuery } from '@tanstack/react-query'

import { fetchPeriods } from '../services'

export const PERIODS_QUERY_KEY = ['periods']

export function usePeriods() {
  return useQuery({
    queryKey: PERIODS_QUERY_KEY,
    queryFn: fetchPeriods,
  })
}
