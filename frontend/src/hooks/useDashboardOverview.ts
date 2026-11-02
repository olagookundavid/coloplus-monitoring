import { useQuery } from '@tanstack/react-query'

import { fetchDashboardOverview } from '../services'

export const DASHBOARD_OVERVIEW_QUERY_KEY = ['dashboard-overview']

export function useDashboardOverview(date?: string) {
  return useQuery({
    queryKey: [...DASHBOARD_OVERVIEW_QUERY_KEY, date],
    queryFn: () => fetchDashboardOverview(date),
  })
}
