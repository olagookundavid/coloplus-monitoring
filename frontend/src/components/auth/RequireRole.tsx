import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { usePermissions } from '../../hooks'
import { landingRouteFor } from '../../utils'
import { MainLoader } from '../ui'

export interface RequireRoleProps {
  /** A real backend permission string (from GET /auth/me's permissions[]),
      never a role name — frontend/CLAUDE.md §14: the backend is the
      authority, this is defense in depth so an unauthorized user is
      redirected cleanly rather than shown a broken admin shell. A list is
      "any of" -- e.g. a route both an admin and a supervisor should reach,
      for different, non-overlapping permissions. */
  permission: string | string[]
  children: ReactNode
}

export function RequireRole({ permission, children }: RequireRoleProps) {
  const { status, permissions } = useAuth()
  const { can } = usePermissions()

  if (status === 'loading') {
    return <MainLoader />
  }

  if (status === 'unauthenticated') {
    return <Navigate to={ROUTES.LOGIN} replace />
  }

  // Denied, not signed out -- land on this account's own home, not
  // unconditionally the customer portal (that was the bug: a denial for
  // any role used to dump everyone on ROUTES.HOME regardless of who they
  // actually are).
  if (!can(permission)) {
    return <Navigate to={landingRouteFor(permissions)} replace />
  }

  return <>{children}</>
}
