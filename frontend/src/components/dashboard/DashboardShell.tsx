import type { ReactNode } from 'react'

import { BottomNav } from './BottomNav'
import { DashboardFooter } from './DashboardFooter'
import type { NavItem } from './navTypes'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'

export interface DashboardShellProps {
  nav: NavItem[]
  children: ReactNode
}

// Shell owns chrome, page owns its own DashboardPageHeader — same precedent
// as AuthLayout. Knows nothing about filters, eyebrows or KPI cards.
export function DashboardShell({ nav, children }: DashboardShellProps) {
  return (
    <div className="flex min-h-dvh">
      <Sidebar nav={nav} />
      {/* min-w-0: without it this flex item won't shrink below the content's
          max-content width (e.g. DataTable's min-w-160 table), which
          cascades a 640px+ minimum up through every ancestor and causes
          page-level horizontal scroll — frontend/CLAUDE.md §3. */}
      <div className="flex min-h-dvh min-w-0 flex-1 flex-col">
        <Topbar />
        {/* pb-20 clears the fixed mobile BottomNav (frontend/CLAUDE.md §3) */}
        <div className="flex min-w-0 flex-1 flex-col pb-20 lg:pb-0">
          <main className="min-w-0 flex-1 px-4 py-6 sm:px-6">{children}</main>
          <DashboardFooter />
        </div>
      </div>
      <BottomNav nav={nav} />
    </div>
  )
}
