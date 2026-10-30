import { BellIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import { useLocation } from 'react-router-dom'

import { IconButton } from '../ui'
import { AccountMenu } from './AccountMenu'
import { CUSTOMER_NAV } from './customerNav'
import { TopbarClock } from './TopbarClock'

export function Topbar() {
  const location = useLocation()
  const activeItem = CUSTOMER_NAV.find((item) => item.to === location.pathname)
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
        <div className="relative">
          <IconButton icon={BellIcon} label="Notifications" variant="ghost" />
          <span
            className="pointer-events-none absolute right-2 top-2 h-2 w-2 rounded-full bg-danger"
            aria-hidden="true"
          />
        </div>
        <AccountMenu />
      </div>
    </header>
  )
}
