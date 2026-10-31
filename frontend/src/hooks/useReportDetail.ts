import { useQuery } from '@tanstack/react-query'

import { fetchReportDetail } from '../services'

export const ADMIN_REPORT_QUERY_KEY = ['admin-report']

export function useReportDetail(id: string | null | undefined) {
  return useQuery({
    queryKey: [...ADMIN_REPORT_QUERY_KEY, id],
    queryFn: () => fetchReportDetail(id as string),
    enabled: Boolean(id),
  })
}
