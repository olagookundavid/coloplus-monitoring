import { useMutation, useQueryClient } from '@tanstack/react-query'

import { saveDailyReportDraft, type DailyReportFormInput } from '../services'
import { DASHBOARD_OVERVIEW_QUERY_KEY } from './useDashboardOverview'
import { MY_REPORTS_QUERY_KEY } from './useMyReports'
import { MY_SITES_QUERY_KEY } from './useMySites'
import { ADMIN_REPORT_QUERY_KEY } from './useReportDetail'

export function useSaveDailyReportMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      input,
    }: {
      id: string
      rowVersion: number
      input: DailyReportFormInput
    }) => saveDailyReportDraft(id, rowVersion, input),
    onSuccess: (_report, { id }) => {
      queryClient.invalidateQueries({ queryKey: DASHBOARD_OVERVIEW_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_REPORTS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: MY_SITES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...ADMIN_REPORT_QUERY_KEY, id] })
    },
  })
}
