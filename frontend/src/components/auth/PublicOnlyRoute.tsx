import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { useAuth } from '../../contexts'
import { landingRouteFor } from '../../utils'
import { MainLoader } from '../ui'

export function PublicOnlyRoute({ children }: { children: ReactNode }) {
  const { status, permissions } = useAuth()

  if (status === 'loading') {
    return <MainLoader />
  }

  if (status === 'authenticated') {
    return <Navigate to={landingRouteFor(permissions)} replace />
  }

  return <>{children}</>
}
