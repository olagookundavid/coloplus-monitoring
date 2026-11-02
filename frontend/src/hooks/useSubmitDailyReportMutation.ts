import { useMutation, useQueryClient } from '@tanstack/react-query'

import { submitDailyReport } from '../services'
import { DASHBOARD_OVERVIEW_QUERY_KEY } from './useDashboardOverview'
import { MY_REPORTS_QUERY_KEY } from './useMyReports'
import { MY_SITES_QUERY_KEY } from './useMySites'
import { ADMIN_REPORT_QUERY_KEY } from './useReportDetail'

export function useSubmitDailyReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => submitDailyReport(id),
    onSuccess: (_report, id) => {
      queryClient.invalidateQueries({ queryKey: DASHBOARD_OVERVIEW_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_REPORTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_SITES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...ADMIN_REPORT_QUERY_KEY, id] })
    },
  })
}
