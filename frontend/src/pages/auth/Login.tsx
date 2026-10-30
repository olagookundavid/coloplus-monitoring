import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { AuthLayout } from '../../components/auth'
import { Button, ButtonLink, Field, Input, PasswordInput } from '../../components/ui'
import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { useLoginMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { getFieldErrors, loginSchema, toast, type LoginFormValues } from '../../utils'

const INITIAL_VALUES: LoginFormValues = { email: '', password: '' }

export function Login() {
  const navigate = useNavigate()
  const { setSession } = useAuth()
  const loginMutation = useLoginMutation()

  const [values, setValues] = useState<LoginFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof LoginFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(loginSchema, values)

  function showError(field: keyof LoginFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange(field: keyof LoginFormValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof LoginFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      const session = await loginMutation.mutateAsync(values)
      setSession(session)
      toast.success('Welcome back.')
      navigate(ROUTES.HOME, { replace: true })
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
      // Clearing the password is our doing, not a blank the user left —
      // don't let the empty-field validator immediately flag it as one.
      setValues((current) => ({ ...current, password: '' }))
      setSubmitted(false)
      setTouched((current) => ({ ...current, password: false }))
    }
  }

  return (
    <AuthLayout
      eyebrow="Coloplus Monitoring"
      title="Log in to your account"
      subtitle="Sign in to view site power status, reports and support."
      footer={<p>Need an account? Ask your Coloplus administrator for an invite.</p>}
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

        <Field label="Email address" htmlFor="login-email" error={showError('email')}>
          <Input
            id="login-email"
            name="email"
            type="email"
            autoComplete="email"
            value={values.email}
            invalid={Boolean(showError('email'))}
            onChange={(event) => handleChange('email', event.target.value)}
            onBlur={() => handleBlur('email')}
          />
        </Field>

        <Field label="Password" htmlFor="login-password" error={showError('password')}>
          <PasswordInput
            id="login-password"
            name="password"
            autoComplete="current-password"
            value={values.password}
            invalid={Boolean(showError('password'))}
            onChange={(event) => handleChange('password', event.target.value)}
            onBlur={() => handleBlur('password')}
          />
        </Field>

        <div className="flex justify-end">
          <ButtonLink to={ROUTES.FORGOT_PASSWORD} variant="link" size="sm">
            Forgot password?
          </ButtonLink>
        </div>

        <Button type="submit" fullWidth loading={loginMutation.isPending}>
          Log in
        </Button>
      </form>
    </AuthLayout>
  )
}
