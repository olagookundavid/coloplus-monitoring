import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createDailyReport, type DailyReportFormInput } from '../services'
import { DASHBOARD_OVERVIEW_QUERY_KEY } from './useDashboardOverview'
import { MY_REPORTS_QUERY_KEY } from './useMyReports'
import { MY_SITES_QUERY_KEY } from './useMySites'

export function useCreateDailyReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      siteId,
      reportDate,
      input,
    }: {
      siteId: string
      reportDate: string
      input: DailyReportFormInput
    }) => createDailyReport(siteId, reportDate, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DASHBOARD_OVERVIEW_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_REPORTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_SITES_QUERY_KEY })
    },
  })
}
