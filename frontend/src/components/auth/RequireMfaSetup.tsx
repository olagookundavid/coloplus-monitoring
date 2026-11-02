import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { MainLoader } from '../ui'

// Guards /mfa-setup itself: bounces away once enrollment is already done,
// so the page can't be re-entered and call POST /auth/mfa/setup again —
// that overwrites the pending secret and would desync whatever the user
// is scanning from what the server now expects.
export function RequireMfaSetup({ children }: { children: ReactNode }) {
  const { status, mfaSetupRequired } = useAuth()

  if (status === 'loading') {
    return <MainLoader />
  }

  if (status === 'unauthenticated') {
    return <Navigate to={ROUTES.LOGIN} replace />
  }

  if (!mfaSetupRequired) {
    return <Navigate to={ROUTES.HOME} replace />
  }

  return <>{children}</>
}
