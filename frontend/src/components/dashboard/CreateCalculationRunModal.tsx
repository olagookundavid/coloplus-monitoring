import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { calculationRunDetailPath } from '../../constants'
import { useCreateCalculationRunMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { Button, Field, Input, Modal } from '../ui'

export interface CreateCalculationRunModalProps {
  open: boolean
  onClose: () => void
}

const MAX_LABEL_LENGTH = 120

export function CreateCalculationRunModal({ open, onClose }: CreateCalculationRunModalProps) {
  const mutation = useCreateCalculationRunMutation()
  const navigate = useNavigate()

  const [month, setMonth] = useState('')
  const [region, setRegion] = useState('')
  const [label, setLabel] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  function resetForm() {
    setMonth('')
    setRegion('')
    setLabel('')
    setFormError(null)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError(null)
    if (!month) return

    try {
      const run = await mutation.mutateAsync({
        month,
        region: region.trim() || undefined,
        label: label.trim() || undefined,
      })
      handleClose()
      navigate(calculationRunDetailPath(run.id))
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
    }
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Store a calculation run"
      subtitle="Freezes that month's figures for every site. Running it again always creates a new, separate run."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="create-run-form" loading={mutation.isPending}>
            Store run
          </Button>
        </>
      }
    >
      <form id="create-run-form" noValidate onSubmit={handleSubmit} className="flex flex-col gap-4">
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        <Field label="Month" htmlFor="run-month">
          <Input
            id="run-month"
            type="month"
            value={month}
            onChange={(event) => setMonth(event.target.value)}
            required
          />
        </Field>

        <Field label="Region" htmlFor="run-region" hint="Leave blank for every region.">
          <Input
            id="run-region"
            value={region}
            onChange={(event) => setRegion(event.target.value)}
          />
        </Field>

        <Field label="Label" htmlFor="run-label" hint="Optional, up to 120 characters.">
          <Input
            id="run-label"
            value={label}
            maxLength={MAX_LABEL_LENGTH}
            onChange={(event) => setLabel(event.target.value)}
          />
        </Field>
      </form>
    </Modal>
  )
}
