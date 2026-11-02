import { useMutation, useQueryClient } from '@tanstack/react-query'

import { rejectReport } from '../services'
import { ADMIN_REPORT_QUERY_KEY } from './useReportDetail'
import { ADMIN_REPORTS_QUERY_KEY } from './useAdminReports'

export function useRejectReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => rejectReport(id, reason),
    onSuccess: (_report, { id }) => {
      queryClient.invalidateQueries({ queryKey: ADMIN_REPORTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...ADMIN_REPORT_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: ['pending-approvals'] })
      queryClient.invalidateQueries({ queryKey: ['recent-audit-events'] })
    },
  })
}
