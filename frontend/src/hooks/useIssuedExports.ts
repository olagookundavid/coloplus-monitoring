import { useQuery } from '@tanstack/react-query'

import { fetchIssuedExports } from '../services'

export const ISSUED_EXPORTS_QUERY_KEY = ['issued-exports']

// Fixed high limit + client-side table, same pattern as every other
// unbounded admin list in this codebase (DataTable has no real
// server-pagination props today).
export function useIssuedExports() {
  return useQuery({
    queryKey: ISSUED_EXPORTS_QUERY_KEY,
    queryFn: () => fetchIssuedExports({ limit: 200 }),
  })
}
