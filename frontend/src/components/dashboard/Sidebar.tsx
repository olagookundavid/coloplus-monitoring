import { LifebuoyIcon, LockClosedIcon } from '@heroicons/react/24/outline'
import { Link, useLocation } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { usePermissions } from '../../hooks'
import { ButtonLink } from '../ui'
import type { NavItem } from './navTypes'

export interface SidebarProps {
  nav: NavItem[]
}

export function Sidebar({ nav }: SidebarProps) {
  const location = useLocation()
  const { can } = usePermissions()
  const visibleNav = nav.filter((item) => !item.permission || can(item.permission))

  return (
    <aside className="hidden lg:flex lg:w-65 lg:shrink-0 lg:flex-col lg:border-r lg:border-border lg:bg-surface">
      <div className="px-6 py-6">
        <Link to={ROUTES.HOME} className="inline-flex items-center gap-1.5">
          <span className="text-xl font-bold tracking-tight text-text">
            coloplus<span className="text-orange-500">.</span>
          </span>
        </Link>
        <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-text-muted">
          Customer portal
        </p>
      </div>

      <nav className="flex-1 px-4" aria-label="Primary">
        <p className="px-2 pb-2 text-xs font-semibold uppercase tracking-wider text-text-muted">
          Workspace
        </p>
        <ul className="flex flex-col gap-1">
          {visibleNav.map((item) => {
            const isActive = location.pathname === item.to
            const Icon = item.icon
            return (
              <li key={item.to}>
                <Link
                  to={item.to}
                  aria-current={isActive ? 'page' : undefined}
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-primary-soft text-primary-700'
                      : 'text-text-muted hover:bg-surface-subtle hover:text-text'
                  }`}
                >
                  <Icon className="h-5 w-5 shrink-0" aria-hidden="true" />
                  <span className="flex-1">{item.label}</span>
                  {Boolean(item.badge) && (
                    <span className="rounded-full bg-primary px-2 py-0.5 text-xs font-semibold text-white">
                      {item.badge}
                    </span>
                  )}
                </Link>
              </li>
            )
          })}
        </ul>
      </nav>

      <div className="mx-4 mb-4 rounded-lg border border-border p-4">
        <LifebuoyIcon className="h-5 w-5 text-primary" aria-hidden="true" />
        <p className="mt-2 text-sm font-semibold text-text">A team behind every site.</p>
        <p className="mt-1 text-sm text-text-muted">
          Your Coloplus operations team is here to help.
        </p>
        {/* Routes into Support tickets — the realistic way to reach the team,
            rather than a dead "#" link or a fabricated email address. */}
        <ButtonLink to={ROUTES.CUSTOMER_TICKETS} variant="link" size="sm" className="mt-2">
          Contact support
        </ButtonLink>
      </div>

      <div className="flex items-center gap-3 border-t border-border px-4 py-4">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-danger-50 text-sm font-semibold text-danger-600">
          AT
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-text">Airtel Nigeria</p>
          <p className="truncate text-xs text-text-muted">Customer workspace</p>
        </div>
        <LockClosedIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
      </div>
    </aside>
  )
}
