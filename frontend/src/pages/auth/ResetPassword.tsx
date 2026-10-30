import { useNavigate, useSearchParams } from 'react-router-dom'

import { AuthLayout, SetPasswordForm } from '../../components/auth'
import { ButtonLink } from '../../components/ui'
import { ROUTES } from '../../constants'
import { toast } from '../../utils'

export function ResetPassword() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')

  if (!token) {
    return (
      <AuthLayout
        title="Reset link invalid"
        subtitle="This password reset link is invalid or incomplete. Request a new one."
      >
        <ButtonLink to={ROUTES.FORGOT_PASSWORD} fullWidth>
          Request a new link
        </ButtonLink>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Reset your password" subtitle="Choose a new password for your account.">
      <SetPasswordForm
        token={token}
        submitLabel="Reset password"
        onSuccess={() => {
          toast.success('Password updated — sign in to continue.')
          navigate(ROUTES.LOGIN, { replace: true })
        }}
      />
    </AuthLayout>
  )
}
