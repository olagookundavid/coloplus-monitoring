import { Outlet } from 'react-router-dom'

import { DashboardShell, OPS_NAV } from '../../components/dashboard'

export function OpsLayout() {
  return (
    <DashboardShell nav={OPS_NAV} portalLabel="Admin console">
      <Outlet />
    </DashboardShell>
  )
}
