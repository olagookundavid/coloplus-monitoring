import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { usePermissions } from '../../hooks'
import { Spinner } from '../ui'

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
  const { status } = useAuth()
  const { can } = usePermissions()

  if (status === 'loading') {
    return <Spinner className="mx-auto mt-20 h-6 w-6" />
  }

  if (status === 'unauthenticated' || !can(permission)) {
    return <Navigate to={ROUTES.HOME} replace />
  }

  return <>{children}</>
}
