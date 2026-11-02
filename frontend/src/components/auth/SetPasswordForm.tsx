import { useState, type FormEvent } from 'react'

import { useSetPasswordMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { getFieldErrors, setPasswordSchema, type SetPasswordFormValues } from '../../utils'
import { Button, Field, PasswordInput, PasswordStrengthMeter } from '../ui'

const INITIAL_VALUES: SetPasswordFormValues = { password: '', confirmPassword: '' }

export interface SetPasswordFormProps {
  token: string
  submitLabel: string
  onSuccess: () => void
}

// Shared by the invite-acceptance and password-reset pages: both pages are
// just copy and a token source around this same form, since the backend
// uses one POST /auth/set-password endpoint for both.
export function SetPasswordForm({ token, submitLabel, onSuccess }: SetPasswordFormProps) {
  const mutation = useSetPasswordMutation()

  const [values, setValues] = useState<SetPasswordFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof SetPasswordFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(setPasswordSchema, values)

  function showError(field: keyof SetPasswordFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange(field: keyof SetPasswordFormValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof SetPasswordFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await mutation.mutateAsync({ token, password: values.password })
      onSuccess()
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
    }
  }

  return (
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
        htmlFor="set-password-password"
        hint={
          !showError('password') ? 'At least 12 characters, with a letter and a number.' : undefined
        }
        error={showError('password')}
      >
        <PasswordInput
          id="set-password-password"
          name="password"
          autoComplete="new-password"
          value={values.password}
          invalid={Boolean(showError('password'))}
          onChange={(event) => handleChange('password', event.target.value)}
          onBlur={() => handleBlur('password')}
        />
        {values.password && (
          <div className="mt-2">
            <PasswordStrengthMeter password={values.password} />
          </div>
        )}
      </Field>

      <Field
        label="Confirm password"
        htmlFor="set-password-confirm"
        error={showError('confirmPassword')}
      >
        <PasswordInput
          id="set-password-confirm"
          name="confirmPassword"
          autoComplete="new-password"
          value={values.confirmPassword}
          invalid={Boolean(showError('confirmPassword'))}
          onChange={(event) => handleChange('confirmPassword', event.target.value)}
          onBlur={() => handleBlur('confirmPassword')}
        />
      </Field>

      <Button type="submit" fullWidth loading={mutation.isPending}>
        {submitLabel}
      </Button>
    </form>
  )
}
