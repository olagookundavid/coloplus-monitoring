import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { Spinner } from '../ui'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status, mfaSetupRequired } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return <Spinner className="mx-auto mt-20 h-6 w-6" />
  }

  if (status === 'unauthenticated') {
    return <Navigate to={ROUTES.LOGIN} replace state={{ from: location }} />
  }

  // The backend grants no permissions until a required MFA enrollment is
  // complete — nothing in the permissioned route tree is usable until then.
  if (mfaSetupRequired) {
    return <Navigate to={ROUTES.MFA_SETUP} replace />
  }

  return <>{children}</>
}
