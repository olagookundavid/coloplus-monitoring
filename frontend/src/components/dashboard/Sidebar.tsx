import { useState } from 'react'
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  LifebuoyIcon,
  LockClosedIcon,
} from '@heroicons/react/24/outline'
import { Link, useLocation } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { usePermissions } from '../../hooks'
import { ButtonLink } from '../ui'
import type { NavItem } from './navTypes'

export interface SidebarProps {
  nav: NavItem[]
}

const COLLAPSED_STORAGE_KEY = 'coloplus.sidebar.collapsed'

function readStoredCollapsed(): boolean {
  try {
    return localStorage.getItem(COLLAPSED_STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

export function Sidebar({ nav }: SidebarProps) {
  const location = useLocation()
  const { can } = usePermissions()
  const visibleNav = nav.filter((item) => !item.permission || can(item.permission))
  const [collapsed, setCollapsed] = useState(readStoredCollapsed)

  const toggleCollapsed = () => {
    setCollapsed((prev) => {
      const next = !prev
      try {
        localStorage.setItem(COLLAPSED_STORAGE_KEY, String(next))
      } catch {
        // Private browsing or storage disabled — collapse state just won't
        // persist across reloads, which is a fine fallback.
      }
      return next
    })
  }

  return (
    <aside
      className={`relative hidden lg:flex lg:h-dvh lg:shrink-0 lg:flex-col lg:border-r lg:border-border lg:bg-surface lg:transition-[width] lg:duration-200 ${
        collapsed ? 'lg:w-20' : 'lg:w-65'
      }`}
    >
      {/* Hand-rolled rather than IconButton: needs to float half outside the
          sidebar's own border, not sit inside the 44px tap-target size. */}
      <button
        type="button"
        onClick={toggleCollapsed}
        aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        className="absolute -right-3.5 top-20 z-10 flex h-7 w-7 items-center justify-center rounded-full border border-border bg-surface text-text-muted shadow-sm transition-colors hover:bg-surface-subtle hover:text-text"
      >
        {collapsed ? (
          <ChevronRightIcon className="h-3.5 w-3.5" aria-hidden="true" />
        ) : (
          <ChevronLeftIcon className="h-3.5 w-3.5" aria-hidden="true" />
        )}
      </button>

      <div className={`shrink-0 py-6 ${collapsed ? 'px-3' : 'px-6'}`}>
        <Link
          to={ROUTES.HOME}
          className={`flex items-center ${collapsed ? 'justify-center' : 'gap-1.5'}`}
        >
          {collapsed ? (
            <span className="text-xl font-bold tracking-tight text-text">
              c<span className="text-orange-500">.</span>
            </span>
          ) : (
            <span className="text-xl font-bold tracking-tight text-text">
              coloplus<span className="text-orange-500">.</span>
            </span>
          )}
        </Link>
        {!collapsed && (
          <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-text-muted">
            Customer portal
          </p>
        )}
      </div>

      {/* min-h-0 is required here: a flex child's default min-height is its
          content size, which would stop it from ever shrinking enough to
          scroll even with flex-1 — the vertical twin of the min-w-0 rule in
          DashboardShell. */}
      <nav
        className={`min-h-0 flex-1 overflow-y-auto ${collapsed ? 'px-2' : 'px-4'}`}
        aria-label="Primary"
      >
        {!collapsed && (
          <p className="px-2 pb-2 text-xs font-semibold uppercase tracking-wider text-text-muted">
            Workspace
          </p>
        )}
        <ul className="flex flex-col gap-1">
          {visibleNav.map((item) => {
            const isActive = location.pathname === item.to
            const Icon = item.icon
            return (
              <li key={item.to}>
                <Link
                  to={item.to}
                  aria-current={isActive ? 'page' : undefined}
                  title={collapsed ? item.label : undefined}
                  className={`flex items-center rounded-lg py-2.5 text-sm font-medium transition-colors ${
                    collapsed ? 'justify-center px-2' : 'gap-2.5 px-3'
                  } ${
                    isActive
                      ? 'bg-primary-soft text-primary-700'
                      : 'text-text-muted hover:bg-surface-subtle hover:text-text'
                  }`}
                >
                  <span className="relative shrink-0">
                    <Icon className="h-5 w-5" aria-hidden="true" />
                    {collapsed && Boolean(item.badge) && (
                      <span
                        className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-primary"
                        aria-hidden="true"
                      />
                    )}
                  </span>
                  {!collapsed && (
                    <>
                      <span className="flex-1">{item.label}</span>
                      {Boolean(item.badge) && (
                        <span className="rounded-full bg-primary px-2 py-0.5 text-xs font-semibold text-white">
                          {item.badge}
                        </span>
                      )}
                    </>
                  )}
                </Link>
              </li>
            )
          })}
        </ul>
      </nav>

      {!collapsed && (
        <div className="mx-4 mb-4 shrink-0 rounded-lg border border-border p-4">
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
      )}

      <div
        className={`flex shrink-0 items-center border-t border-border py-4 ${
          collapsed ? 'justify-center px-2' : 'gap-3 px-4'
        }`}
      >
        <span
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-danger-50 text-sm font-semibold text-danger-600"
          title={collapsed ? 'Airtel Nigeria' : undefined}
        >
          AT
        </span>
        {!collapsed && (
          <>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-text">Airtel Nigeria</p>
              <p className="truncate text-xs text-text-muted">Customer workspace</p>
            </div>
            <LockClosedIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
          </>
        )}
      </div>
    </aside>
  )
}
