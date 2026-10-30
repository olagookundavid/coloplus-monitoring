import { Link, useLocation } from 'react-router-dom'

import { usePermissions } from '../../hooks'
import type { NavItem } from './navTypes'

export interface BottomNavProps {
  nav: NavItem[]
}

export function BottomNav({ nav }: BottomNavProps) {
  const location = useLocation()
  const { can } = usePermissions()
  const visibleNav = nav.filter((item) => !item.permission || can(item.permission))

  return (
    <nav
      aria-label="Primary"
      className="fixed inset-x-0 bottom-0 z-40 flex border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] lg:hidden"
    >
      {visibleNav.map((item) => {
        const isActive = location.pathname === item.to
        const Icon = item.icon
        return (
          <Link
            key={item.to}
            to={item.to}
            aria-current={isActive ? 'page' : undefined}
            className={`relative flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 px-1 py-2 text-xs font-medium ${
              isActive ? 'text-primary-700' : 'text-text-muted'
            }`}
          >
            <Icon className="h-5 w-5" aria-hidden="true" />
            <span className="w-full truncate text-center">{item.label}</span>
            {Boolean(item.badge) && (
              <span
                className="absolute right-1/4 top-1.5 h-2 w-2 rounded-full bg-danger"
                aria-hidden="true"
              />
            )}
          </Link>
        )
      })}
    </nav>
  )
}
