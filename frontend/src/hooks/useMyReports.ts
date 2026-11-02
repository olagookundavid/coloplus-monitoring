import { useQuery } from '@tanstack/react-query'

import { fetchReports } from '../services'
import type { ReportStatus } from '../types'

export const MY_REPORTS_QUERY_KEY = ['my-reports']

// The reporter's own report history, any status or across all dates --
// backs Daily reports, Corrections and the Corrections nav badge.
export function useMyReports(
  params: { status?: ReportStatus; limit?: number; enabled?: boolean } = {}
) {
  return useQuery({
    queryKey: [...MY_REPORTS_QUERY_KEY, params.status, params.limit],
    queryFn: () => fetchReports({ mine: true, status: params.status, limit: params.limit }),
    enabled: params.enabled ?? true,
  })
}
