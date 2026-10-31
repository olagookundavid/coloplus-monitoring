import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { AuthLayout } from '../../components/auth'
import {
  Button,
  ButtonLink,
  Checkbox,
  Field,
  Input,
  OtpInput,
  PasswordInput,
} from '../../components/ui'
import { ROUTES } from '../../constants'
import { useAuth } from '../../contexts'
import { useLoginMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import {
  getFieldErrors,
  landingRouteFor,
  loginSchema,
  mfaCodeSchema,
  toast,
  type LoginFormValues,
  type MfaCodeFormValues,
} from '../../utils'

const INITIAL_MFA_VALUES: MfaCodeFormValues = { code: '' }

interface LocationState {
  from?: { pathname: string }
}

// "Keep me signed in on this device" only remembers the *email*, for
// autofill convenience — never the password. Storing a password in
// localStorage would be readable by any XSS on the page; the browser's own
// password manager already handles that part securely (autoComplete
// "current-password" on the field below is what lets it offer to).
const REMEMBERED_EMAIL_KEY = 'coloplus.auth.rememberedEmail'

function readRememberedEmail(): string {
  try {
    return localStorage.getItem(REMEMBERED_EMAIL_KEY) ?? ''
  } catch {
    return ''
  }
}

function writeRememberedEmail(email: string | null): void {
  try {
    if (email) {
      localStorage.setItem(REMEMBERED_EMAIL_KEY, email)
    } else {
      localStorage.removeItem(REMEMBERED_EMAIL_KEY)
    }
  } catch {
    // Private browsing or storage disabled — just won't be remembered.
  }
}

export function Login() {
  const navigate = useNavigate()
  const location = useLocation()
  const { refreshSession } = useAuth()
  const loginMutation = useLoginMutation()

  const [step, setStep] = useState<'credentials' | 'mfa'>('credentials')
  const [values, setValues] = useState<LoginFormValues>(() => ({
    email: readRememberedEmail(),
    password: '',
  }))
  const [touched, setTouched] = useState<Partial<Record<keyof LoginFormValues, boolean>>>({})
  const [mfaValues, setMfaValues] = useState<MfaCodeFormValues>(INITIAL_MFA_VALUES)
  const [mfaTouched, setMfaTouched] = useState(false)
  const [useRecoveryCode, setUseRecoveryCode] = useState(false)
  const [rememberMe, setRememberMe] = useState(() => Boolean(readRememberedEmail()))
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(loginSchema, values)
  const mfaFieldErrors = getFieldErrors(mfaCodeSchema, mfaValues)

  function showError(field: keyof LoginFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange(field: keyof LoginFormValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof LoginFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  // Respects an explicit redirect target (RequireAuth/RequireRole sent them
  // here from somewhere specific) over the permission-based default, so e.g.
  // an admin bounced off /ops/users for being signed out lands back there.
  // ROUTES.HOME doesn't count as "somewhere specific": RequireAuth records
  // it as `from` for every signed-out visitor who just opens the app at
  // '/', not only ones deep-linking into the customer portal, so treating
  // it as deliberate would make the permission-based default never apply.
  function destinationAfterLogin(permissions: string[]): string {
    const from = (location.state as LocationState | null)?.from?.pathname
    if (from && from !== ROUTES.HOME) return from
    return landingRouteFor(permissions)
  }

  // Shared by both steps: login is one call resubmitted whole with
  // `totpCode` filled in, not a separate "verify" endpoint.
  async function completeLogin(totpCode?: string) {
    const result = await loginMutation.mutateAsync({ ...values, totpCode })
    if (result.status === 'mfa_required') {
      setStep('mfa')
      setMfaValues(INITIAL_MFA_VALUES)
      setMfaTouched(false)
      return
    }

    const session = await refreshSession()
    // Not actually in the dashboard yet if MFA enrollment is still required —
    // the welcome toast belongs at the point they're really authenticated in.
    if (!result.mfaSetupRequired) {
      toast.success('Welcome back.')
    }
    navigate(
      result.mfaSetupRequired
        ? ROUTES.MFA_SETUP
        : destinationAfterLogin(session?.permissions ?? []),
      { replace: true }
    )
  }

  async function handleCredentialsSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await completeLogin()
      // Reaching here means the email and password were both correct
      // (whether or not an MFA step follows) — the right moment to
      // remember (or forget) the email for next time.
      writeRememberedEmail(rememberMe ? values.email.trim() : null)
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

  async function submitMfaCode(code: string) {
    setFormError(null)
    try {
      await completeLogin(code)
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
      setMfaValues(INITIAL_MFA_VALUES)
      setMfaTouched(false)
    }
  }

  async function handleMfaSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMfaTouched(true)
    if (Object.keys(mfaFieldErrors).length > 0) return
    await submitMfaCode(mfaValues.code)
  }

  function backToCredentials() {
    setStep('credentials')
    setValues((current) => ({ ...current, password: '' }))
    setMfaValues(INITIAL_MFA_VALUES)
    setUseRecoveryCode(false)
    setFormError(null)
    setSubmitted(false)
    setMfaTouched(false)
  }

  function toggleRecoveryCode() {
    setUseRecoveryCode((current) => !current)
    setMfaValues(INITIAL_MFA_VALUES)
    setMfaTouched(false)
    setFormError(null)
  }

  if (step === 'mfa') {
    return (
      <AuthLayout
        title="Verify it's you"
        subtitle={
          useRecoveryCode
            ? 'Enter one of your recovery codes.'
            : 'Enter the 6-digit code from your authenticator app.'
        }
        footer={
          <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-center">
            <Button variant="link" size="sm" onClick={toggleRecoveryCode}>
              {useRecoveryCode
                ? 'Enter a 6-digit code instead'
                : 'Lost your device? Use a recovery code'}
            </Button>
            <span className="text-text-muted">&middot;</span>
            <Button variant="link" size="sm" onClick={backToCredentials}>
              Back to sign in
            </Button>
          </div>
        }
      >
        <form noValidate onSubmit={handleMfaSubmit} className="flex flex-col gap-4">
          {formError && (
            <div
              role="alert"
              className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
            >
              {formError}
            </div>
          )}

          <Field
            label="Verification code"
            htmlFor="login-mfa-code"
            error={mfaTouched ? mfaFieldErrors.code : undefined}
          >
            {useRecoveryCode ? (
              <Input
                id="login-mfa-code"
                name="code"
                placeholder="XXXX-XXXX-XXXX"
                autoComplete="one-time-code"
                value={mfaValues.code}
                invalid={Boolean(mfaTouched && mfaFieldErrors.code)}
                onChange={(event) => setMfaValues({ code: event.target.value })}
                onBlur={() => setMfaTouched(true)}
              />
            ) : (
              <OtpInput
                id="login-mfa-code"
                value={mfaValues.code}
                invalid={Boolean(mfaTouched && mfaFieldErrors.code)}
                autoFocus
                onChange={(code) => setMfaValues({ code })}
                onComplete={(code) => void submitMfaCode(code)}
              />
            )}
          </Field>

          <Button type="submit" fullWidth loading={loginMutation.isPending}>
            Verify
          </Button>
        </form>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Log in to your account"
      footer={
        <p>
          Accounts are created by a Coloplus administrator. Need access? Contact your site
          administrator.
        </p>
      }
    >
      <form noValidate onSubmit={handleCredentialsSubmit} className="flex flex-col gap-4">
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

        <div>
          <div className="mb-1.5 flex items-center justify-between gap-2">
            <label htmlFor="login-password" className="text-sm font-medium text-text">
              Password
            </label>
            <ButtonLink to={ROUTES.FORGOT_PASSWORD} variant="link" size="sm">
              Forgot password?
            </ButtonLink>
          </div>
          <Field htmlFor="login-password" error={showError('password')}>
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
        </div>

        <Checkbox
          label="Keep me signed in on this device"
          checked={rememberMe}
          onChange={(event) => setRememberMe(event.target.checked)}
        />

        <Button type="submit" fullWidth loading={loginMutation.isPending}>
          Log in
        </Button>
      </form>
    </AuthLayout>
  )
}
