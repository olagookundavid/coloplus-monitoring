import { useQuery } from '@tanstack/react-query'

import { fetchCalculationRunComparison } from '../services'

export const CALCULATION_RUN_COMPARISON_QUERY_KEY = ['calculation-run-comparison']

export function useCalculationRunComparison(id: string | undefined) {
  return useQuery({
    queryKey: [...CALCULATION_RUN_COMPARISON_QUERY_KEY, id],
    queryFn: () => fetchCalculationRunComparison(id as string),
    enabled: Boolean(id),
  })
}
