import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { MainLoader } from '../ui'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status, mfaSetupRequired } = useAuth()

  if (status === 'loading') {
    return <MainLoader />
  }

  if (status === 'unauthenticated') {
    return <Navigate to={ROUTES.LOGIN} replace />
  }

  // The backend grants no permissions until a required MFA enrollment is
  // complete — nothing in the permissioned route tree is usable until then.
  if (mfaSetupRequired) {
    return <Navigate to={ROUTES.MFA_SETUP} replace />
  }

  return <>{children}</>
}
