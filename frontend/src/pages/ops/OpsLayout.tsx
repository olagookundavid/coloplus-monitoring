import { Outlet } from 'react-router-dom'

import { buildOpsNavGroups, DashboardShell } from '../../components/dashboard'
import { useMyReports, usePermissions } from '../../hooks'

export function OpsLayout() {
  // A supervisor reaches this layout too (Approvals, reports.review) without
  // holding users.manage -- "Admin portal" would misdescribe their view.
  // A field reporter holds neither.
  const { can } = usePermissions()
  const portalLabel = can('users.manage')
    ? 'Admin portal'
    : can('reports.review')
      ? 'Operations console'
      : can('reports.create')
        ? 'Field reporting portal'
        : 'Operations console'

  // Only field reporters see the Corrections badge; fetching their own
  // rejected-report count for anyone else would be a wasted, permission-
  // denied request.
  const isReporter = can('reports.create')
  const correctionsQuery = useMyReports({ status: 'rejected', limit: 1, enabled: isReporter })
  const correctionsBadge = isReporter ? (correctionsQuery.data?.total ?? 0) : 0

  return (
    <DashboardShell nav={buildOpsNavGroups(correctionsBadge)} portalLabel={portalLabel}>
      <Outlet />
    </DashboardShell>
  )
}
