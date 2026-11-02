import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { toDataURL } from 'qrcode'

import { AuthLayout, RecoveryCodesModal } from '../../components/auth'
import { Button, Field, OtpInput, Spinner } from '../../components/ui'
import { useAuth } from '../../contexts'
import { useEnableMfaMutation, useMfaSetupMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import {
  getFieldErrors,
  landingRouteFor,
  mfaCodeSchema,
  toast,
  type MfaCodeFormValues,
} from '../../utils'

const INITIAL_VALUES: MfaCodeFormValues = { code: '' }

export function MfaSetup() {
  const navigate = useNavigate()
  const { refreshSession } = useAuth()
  const setupMutation = useMfaSetupMutation()
  const enableMutation = useEnableMfaMutation()

  // POST /auth/mfa/setup overwrites the pending secret on every call, and
  // StrictMode double-invokes mount effects in dev — call it exactly once.
  const started = useRef(false)

  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null)
  const [secret, setSecret] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [values, setValues] = useState<MfaCodeFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null)
  const [alreadyEnabled, setAlreadyEnabled] = useState(false)
  const [continuing, setContinuing] = useState(false)

  const fieldErrors = getFieldErrors(mfaCodeSchema, values)
  const codeError = touched || submitted ? fieldErrors.code : undefined

  useEffect(() => {
    if (started.current) return
    started.current = true

    async function start() {
      try {
        const result = await setupMutation.mutateAsync()
        setSecret(result.secret)
        setQrDataUrl(await toDataURL(result.otpauthUrl))
      } catch (error) {
        setLoadError(
          error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
        )
      }
    }
    void start()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deliberately runs once, see `started`
  }, [])

  async function submitCode(code: string) {
    setFormError(null)
    try {
      const result = await enableMutation.mutateAsync({ code })
      if (result.recoveryCodes) {
        setRecoveryCodes(result.recoveryCodes)
      } else {
        setAlreadyEnabled(true)
      }
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
      setValues(INITIAL_VALUES)
      setSubmitted(false)
      setTouched(false)
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    if (Object.keys(fieldErrors).length > 0) return
    await submitCode(values.code)
  }

  async function handleContinue() {
    setContinuing(true)
    const session = await refreshSession()
    toast.success('Welcome back.')
    navigate(landingRouteFor(session?.permissions ?? []), { replace: true })
  }

  if (alreadyEnabled) {
    return (
      <AuthLayout
        title="Two-step verification is already on"
        subtitle="Your account already has two-step verification enabled."
      >
        <Button fullWidth onClick={handleContinue} loading={continuing}>
          Continue
        </Button>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Set up two-step verification"
      subtitle="Your role requires an authenticator app for extra account security."
    >
      <div className="flex flex-col gap-4">
        {loadError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {loadError}
          </div>
        )}

        {!qrDataUrl && !loadError && <Spinner className="mx-auto h-6 w-6" />}

        {qrDataUrl && (
          <>
            <div className="flex flex-col items-center gap-3 rounded-lg border border-border bg-surface-subtle p-4">
              <img src={qrDataUrl} alt="Scan with your authenticator app" className="h-40 w-40" />
              {secret && (
                <p className="break-all text-center font-mono text-xs text-text-muted">{secret}</p>
              )}
            </div>

            <form noValidate onSubmit={handleSubmit} className="flex flex-col gap-4">
              {formError && (
                <div
                  role="alert"
                  className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
                >
                  {formError}
                </div>
              )}

              <Field label="Verification code" htmlFor="mfa-setup-code" error={codeError}>
                <OtpInput
                  id="mfa-setup-code"
                  value={values.code}
                  invalid={Boolean(codeError)}
                  onChange={(code) => setValues({ code })}
                  onComplete={(code) => void submitCode(code)}
                />
              </Field>

              <Button type="submit" fullWidth loading={enableMutation.isPending}>
                Enable two-step verification
              </Button>
            </form>
          </>
        )}
      </div>

      <RecoveryCodesModal
        open={recoveryCodes !== null}
        codes={recoveryCodes ?? []}
        onContinue={handleContinue}
        continuing={continuing}
      />
    </AuthLayout>
  )
}
