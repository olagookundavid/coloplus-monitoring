import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { AuthLayout } from '../../components/auth'
import { Button, ButtonLink, Field, PasswordInput } from '../../components/ui'
import { ROUTES } from '../../constants'
import { useAcceptInviteMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { acceptInviteSchema, getFieldErrors, toast, type AcceptInviteFormValues } from '../../utils'

const INITIAL_VALUES: AcceptInviteFormValues = { password: '', confirmPassword: '' }

export function AcceptInvite() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const mutation = useAcceptInviteMutation()

  const [values, setValues] = useState<AcceptInviteFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof AcceptInviteFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(acceptInviteSchema, values)

  function showError(field: keyof AcceptInviteFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange(field: keyof AcceptInviteFormValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof AcceptInviteFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token) return
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await mutation.mutateAsync({ token, password: values.password })
      toast.success('Your account is active — sign in to continue.')
      navigate(ROUTES.LOGIN, { replace: true })
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
    }
  }

  if (!token) {
    return (
      <AuthLayout
        eyebrow="Coloplus Monitoring"
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
    <AuthLayout
      eyebrow="Coloplus Monitoring"
      title="Set your password"
      subtitle="Set a password to activate your account."
    >
      <form noValidate onSubmit={handleSubmit} className="flex flex-col gap-4">
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        <Field
          label="New password"
          htmlFor="invite-password"
          hint={!showError('password') ? 'Minimum 8 characters.' : undefined}
          error={showError('password')}
        >
          <PasswordInput
            id="invite-password"
            name="password"
            autoComplete="new-password"
            value={values.password}
            invalid={Boolean(showError('password'))}
            onChange={(event) => handleChange('password', event.target.value)}
            onBlur={() => handleBlur('password')}
          />
        </Field>

        <Field
          label="Confirm password"
          htmlFor="invite-confirm-password"
          error={showError('confirmPassword')}
        >
          <PasswordInput
            id="invite-confirm-password"
            name="confirmPassword"
            autoComplete="new-password"
            value={values.confirmPassword}
            invalid={Boolean(showError('confirmPassword'))}
            onChange={(event) => handleChange('confirmPassword', event.target.value)}
            onBlur={() => handleBlur('confirmPassword')}
          />
        </Field>

        <Button type="submit" fullWidth loading={mutation.isPending}>
          Activate account
        </Button>
      </form>
    </AuthLayout>
  )
}
