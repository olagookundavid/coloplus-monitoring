import { useQuery } from '@tanstack/react-query'

import { fetchCalculationRun } from '../services'

export const CALCULATION_RUN_QUERY_KEY = ['calculation-run']

export function useCalculationRunDetail(id: string | undefined) {
  return useQuery({
    queryKey: [...CALCULATION_RUN_QUERY_KEY, id],
    queryFn: () => fetchCalculationRun(id as string),
    enabled: Boolean(id),
  })
}
