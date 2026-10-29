import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'

export function PublicOnlyRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth()

  if (isAuthenticated) {
    return <Navigate to={ROUTES.HOME} replace />
  }

  return <>{children}</>
}
