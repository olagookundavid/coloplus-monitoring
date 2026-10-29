import { useState, type FormEvent } from 'react'

import { AuthLayout } from '../../components/auth'
import { Button, ButtonLink, Field, Input } from '../../components/ui'
import { ROUTES } from '../../constants'
import { useForgotPasswordMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { forgotPasswordSchema, getFieldErrors, type ForgotPasswordFormValues } from '../../utils'

const INITIAL_VALUES: ForgotPasswordFormValues = { email: '' }

export function ForgotPassword() {
  const mutation = useForgotPasswordMutation()

  const [values, setValues] = useState<ForgotPasswordFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [sentFor, setSentFor] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(forgotPasswordSchema, values)
  const emailError = touched || submitted ? fieldErrors.email : undefined

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await mutation.mutateAsync(values)
      setSentFor(values.email.trim())
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
    }
  }

  if (sentFor) {
    return (
      <AuthLayout
        eyebrow="Coloplus Monitoring"
        title="Check your email"
        subtitle={`If an account exists for ${sentFor}, we've sent a link to reset your password.`}
      >
        <div className="flex flex-col gap-3">
          <ButtonLink to={ROUTES.LOGIN} fullWidth>
            Back to log in
          </ButtonLink>
          <Button variant="ghost" fullWidth onClick={() => setSentFor(null)}>
            Try a different email
          </Button>
        </div>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      eyebrow="Coloplus Monitoring"
      title="Reset your password"
      subtitle="Enter your email and we'll send you a link to reset your password."
      footer={
        <ButtonLink to={ROUTES.LOGIN} variant="link" size="sm">
          Back to log in
        </ButtonLink>
      }
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

        <Field label="Email address" htmlFor="forgot-email" error={emailError}>
          <Input
            id="forgot-email"
            name="email"
            type="email"
            autoComplete="email"
            value={values.email}
            invalid={Boolean(emailError)}
            onChange={(event) => setValues({ email: event.target.value })}
            onBlur={() => setTouched(true)}
          />
        </Field>

        <Button type="submit" fullWidth loading={mutation.isPending}>
          Send reset link
        </Button>
      </form>
    </AuthLayout>
  )
}
