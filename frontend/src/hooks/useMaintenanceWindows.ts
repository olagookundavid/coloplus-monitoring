import { useQuery } from '@tanstack/react-query'

import { fetchMaintenanceWindows } from '../services'
import type { MaintenanceFilter } from '../types'

export const MAINTENANCE_QUERY_KEY = ['maintenance']

// Fixed high limit + full client-side table, same pattern as EmailLog.tsx/
// AuditLog.tsx -- DataTable has no real server-pagination props today.
export function useMaintenanceWindows(filter: MaintenanceFilter = {}, enabled = true) {
  const resolved: MaintenanceFilter = { limit: 200, ...filter }
  return useQuery({
    queryKey: [...MAINTENANCE_QUERY_KEY, resolved],
    queryFn: () => fetchMaintenanceWindows(resolved),
    enabled,
  })
}
