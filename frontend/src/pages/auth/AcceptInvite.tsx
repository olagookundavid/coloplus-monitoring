import { useNavigate, useSearchParams } from 'react-router-dom'

import { AuthLayout, SetPasswordForm } from '../../components/auth'
import { ButtonLink } from '../../components/ui'
import { ROUTES } from '../../constants'
import { toast } from '../../utils'

export function AcceptInvite() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')

  if (!token) {
    return (
      <AuthLayout
        title="Invite link invalid"
        subtitle="This invite link is invalid or incomplete. Ask your administrator to resend it."
      >
        <ButtonLink to={ROUTES.LOGIN} fullWidth>
          Back to log in
        </ButtonLink>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Set your password" subtitle="Set a password to activate your account.">
      <SetPasswordForm
        token={token}
        submitLabel="Activate account"
        onSuccess={() => {
          toast.success('Your account is active — sign in to continue.')
          navigate(ROUTES.LOGIN, { replace: true })
        }}
      />
    </AuthLayout>
  )
}
