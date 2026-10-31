import { useMutation, useQueryClient } from '@tanstack/react-query'

import { approveReport } from '../services'
import { ADMIN_REPORT_QUERY_KEY } from './useReportDetail'
import { ADMIN_REPORTS_QUERY_KEY } from './useAdminReports'

export function useApproveReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, comment }: { id: string; comment?: string }) => approveReport(id, comment),
    onSuccess: (_report, { id }) => {
      queryClient.invalidateQueries({ queryKey: ADMIN_REPORTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...ADMIN_REPORT_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: ['pending-approvals'] })
      queryClient.invalidateQueries({ queryKey: ['recent-audit-events'] })
    },
  })
}
