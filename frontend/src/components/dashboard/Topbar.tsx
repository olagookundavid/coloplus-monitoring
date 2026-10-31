import { ChevronRightIcon } from '@heroicons/react/24/outline'
import { useLocation } from 'react-router-dom'

import { AccountMenu } from './AccountMenu'
import type { NavGroup } from './navTypes'
import { NotificationsMenu } from './NotificationsMenu'
import { TopbarClock } from './TopbarClock'

export interface TopbarProps {
  nav: NavGroup[]
}

export function Topbar({ nav }: TopbarProps) {
  const location = useLocation()
  const activeItem = nav
    .flatMap((group) => group.items)
    .find((item) => item.to === location.pathname)
  const breadcrumbLabel = activeItem?.label ?? 'Overview'

  return (
    <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border bg-surface px-4 sm:px-6">
      <div className="flex min-w-0 items-center gap-1.5 text-sm text-text-muted">
        <span>Workspace</span>
        <ChevronRightIcon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        <span className="truncate font-medium text-text">{breadcrumbLabel}</span>
      </div>

      <div className="flex shrink-0 items-center gap-3">
        <TopbarClock />
        <span className="hidden h-8 w-px bg-border md:block" aria-hidden="true" />
        <NotificationsMenu />
        <AccountMenu />
      </div>
    </header>
  )
}
