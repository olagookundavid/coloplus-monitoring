import type { ReactNode } from 'react'

import { BottomNav } from './BottomNav'
import { DashboardFooter } from './DashboardFooter'
import type { NavItem } from './navTypes'
import { Sidebar, type SidebarOrgBadge, type SidebarSupportCard } from './Sidebar'
import { Topbar } from './Topbar'

export interface DashboardShellProps {
  nav: NavItem[]
  /** Which portal this is — e.g. "Customer portal," "Admin console." */
  portalLabel: string
  orgBadge?: SidebarOrgBadge
  supportCard?: SidebarSupportCard
  children: ReactNode
}

// The one shell for every portal/role: it knows nothing about a specific
// customer or role beyond the identity it's handed here. A page owns its
// own DashboardPageHeader — same precedent as AuthLayout. Knows nothing
// about filters, eyebrows or KPI cards.
export function DashboardShell({
  nav,
  portalLabel,
  orgBadge,
  supportCard,
  children,
}: DashboardShellProps) {
  return (
    // h-dvh + overflow-hidden pins the whole shell to exactly the viewport
    // height — without it, the root grows with page content and the
    // Sidebar/Topbar scroll away with everything else. Sidebar and Topbar
    // stay fixed; only the content region below scrolls (overflow-y-auto).
    <div className="flex h-dvh overflow-hidden">
      <Sidebar nav={nav} portalLabel={portalLabel} orgBadge={orgBadge} supportCard={supportCard} />
      {/* min-w-0: without it this flex item won't shrink below the content's
          max-content width (e.g. DataTable's min-w-160 table), which
          cascades a 640px+ minimum up through every ancestor and causes
          page-level horizontal scroll — frontend/CLAUDE.md §3. */}
      <div className="flex h-dvh min-w-0 flex-1 flex-col">
        <Topbar nav={nav} />
        {/* pb-20 clears the fixed mobile BottomNav (frontend/CLAUDE.md §3).
            This is the one scrollable region in the whole shell. */}
        <div className="min-w-0 flex-1 overflow-y-auto pb-20 lg:pb-0">
          <main className="min-w-0 px-4 py-6 sm:px-6">{children}</main>
          <DashboardFooter />
        </div>
      </div>
      <BottomNav nav={nav} />
    </div>
  )
}
