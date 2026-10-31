import { Outlet } from 'react-router-dom'

import { DashboardShell, OPS_NAV_GROUPS } from '../../components/dashboard'
import { usePermissions } from '../../hooks'

export function OpsLayout() {
  // A supervisor reaches this layout too (Approvals, reports.review) without
  // holding users.manage -- "Admin console" would misdescribe their view.
  const { can } = usePermissions()
  const portalLabel = can('users.manage') ? 'Admin console' : 'Operations console'

  return (
    <DashboardShell nav={OPS_NAV_GROUPS} portalLabel={portalLabel}>
      <Outlet />
    </DashboardShell>
  )
}
