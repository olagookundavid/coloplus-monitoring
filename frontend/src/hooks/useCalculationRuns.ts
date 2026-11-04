import { useQuery } from '@tanstack/react-query'

import { fetchCalculationRuns } from '../services'

export const CALCULATION_RUNS_QUERY_KEY = ['calculation-runs']

// Fixed high limit + client-side table, same pattern as every other
// unbounded admin list in this codebase (DataTable has no server-pagination
// props today).
export function useCalculationRuns() {
  return useQuery({
    queryKey: CALCULATION_RUNS_QUERY_KEY,
    queryFn: () => fetchCalculationRuns({ limit: 200 }),
  })
}
