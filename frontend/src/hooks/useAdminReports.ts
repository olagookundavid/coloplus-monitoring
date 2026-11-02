import { useQuery } from '@tanstack/react-query'

import { fetchReports } from '../services'
import type { ReportStatus } from '../types'

export const ADMIN_REPORTS_QUERY_KEY = ['admin-reports']

export function useAdminReports(status: ReportStatus) {
  return useQuery({
    queryKey: [...ADMIN_REPORTS_QUERY_KEY, status],
    queryFn: () => fetchReports({ status, limit: 200 }),
  })
}
