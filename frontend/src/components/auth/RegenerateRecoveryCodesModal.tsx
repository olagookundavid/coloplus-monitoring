import { useState, type FormEvent } from 'react'

import { useRegenerateRecoveryCodesMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { getFieldErrors, mfaCodeSchema, type MfaCodeFormValues } from '../../utils'
import { Button, Field, Modal, OtpInput } from '../ui'
import { RecoveryCodesModal } from './RecoveryCodesModal'

const INITIAL_VALUES: MfaCodeFormValues = { code: '' }

export interface RegenerateRecoveryCodesModalProps {
  open: boolean
  onClose: () => void
}

export function RegenerateRecoveryCodesModal({ open, onClose }: RegenerateRecoveryCodesModalProps) {
  const mutation = useRegenerateRecoveryCodesMutation()

  const [values, setValues] = useState<MfaCodeFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [newCodes, setNewCodes] = useState<string[] | null>(null)

  const fieldErrors = getFieldErrors(mfaCodeSchema, values)
  const codeError = touched || submitted ? fieldErrors.code : undefined

  function reset() {
    setValues(INITIAL_VALUES)
    setTouched(false)
    setSubmitted(false)
    setFormError(null)
    setNewCodes(null)
  }

  function handleClose() {
    onClose()
    reset()
  }

  async function submitCode(code: string) {
    setFormError(null)
    try {
      const result = await mutation.mutateAsync({ code })
      setNewCodes(result.recoveryCodes ?? [])
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

  if (newCodes !== null) {
    return <RecoveryCodesModal open codes={newCodes} onContinue={handleClose} />
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Regenerate recovery codes"
      subtitle="Enter a current code from your authenticator app. Your old recovery codes stop working immediately."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="regenerate-recovery-codes-form" loading={mutation.isPending}>
            Generate new codes
          </Button>
        </>
      }
    >
      <form
        id="regenerate-recovery-codes-form"
        noValidate
        onSubmit={handleSubmit}
        className="flex flex-col gap-4"
      >
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        <Field label="Verification code" htmlFor="regenerate-recovery-codes-code" error={codeError}>
          <OtpInput
            id="regenerate-recovery-codes-code"
            value={values.code}
            invalid={Boolean(codeError)}
            onChange={(code) => setValues({ code })}
            onComplete={(code) => void submitCode(code)}
          />
        </Field>
      </form>
    </Modal>
  )
}
