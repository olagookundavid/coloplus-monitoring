import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { MainLoader } from '../ui'

export function PublicOnlyRoute({ children }: { children: ReactNode }) {
  const { status, user } = useAuth()

  if (status === 'loading') {
    return <MainLoader />
  }

  if (status === 'authenticated') {
    return <Navigate to={user?.role === 'admin' ? ROUTES.OPS_OVERVIEW : ROUTES.HOME} replace />
  }

  return <>{children}</>
}
