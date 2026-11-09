import type { ReactNode } from 'react'
import { ChevronRightIcon } from '@heroicons/react/24/outline'
import { useLocation } from 'react-router-dom'

import { Logo } from '../icons'
import { AccountMenu } from './AccountMenu'
import type { NavGroup } from './navTypes'
import { NotificationsMenu } from './NotificationsMenu'
import { TopbarClock } from './TopbarClock'

export interface TopbarProps {
  nav: NavGroup[]
  /** The signed-in customer's own co-brand mark (e.g. a customer logo) — only the
      customer portal passes this; ops portals have no customer to brand for. */
  brandMark?: ReactNode
}

export function Topbar({ nav, brandMark }: TopbarProps) {
  const location = useLocation()
  const activeItem = nav
    .flatMap((group) => group.items)
    .find((item) => item.to === location.pathname)
  const breadcrumbLabel = activeItem?.label ?? 'Overview'

  return (
    <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border bg-surface px-4 sm:px-6">
      <div className="flex min-w-0 items-center gap-2">
        {/* The sidebar already carries the brand mark at lg+; below that it's
            hidden entirely (frontend/CLAUDE.md §3), so this is the only
            branding a phone-width user ever sees. */}
        <Logo variant="mark" tone="black" className="h-7 w-7 shrink-0 lg:hidden" />
        <div className="flex min-w-0 items-center gap-1.5 text-sm text-text-muted">
          <span className="hidden lg:inline">Workspace</span>
          <ChevronRightIcon className="hidden h-3.5 w-3.5 shrink-0 lg:block" aria-hidden="true" />
          <span className="truncate font-medium text-text">{breadcrumbLabel}</span>
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-3">
        <TopbarClock />
        <span className="hidden h-8 w-px bg-border md:block" aria-hidden="true" />
        <NotificationsMenu />
        <AccountMenu />
        {brandMark && (
          <>
            <span className="hidden h-8 w-px bg-border md:block" aria-hidden="true" />
            {brandMark}
          </>
        )}
      </div>
    </header>
  )
}
