import { useQuery } from '@tanstack/react-query'

import { fetchReports } from '../services'

// Resolves the single approved report for a site and date, for the Request
// a correction modal's report path -- a report is identified by site + date,
// not a ready-made picker list, so this is a narrow, on-demand lookup rather
// than a cached list like useMyReports.
export function useApprovedReportForDate(siteId: string, date: string) {
  return useQuery({
    queryKey: ['approved-report-for-date', siteId, date],
    queryFn: async () => {
      const { reports } = await fetchReports({ siteId, from: date, to: date, status: 'approved', limit: 1 })
      return reports[0] ?? null
    },
    enabled: Boolean(siteId && date),
  })
}
