import { useState, type FormEvent } from 'react'

import { useClosePeriodMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { toast } from '../../utils'
import { Button, Field, Input, Modal } from '../ui'

export interface ClosePeriodModalProps {
  open: boolean
  onClose: () => void
}

// There's no "closeable months" list from the backend -- GET /periods only
// returns months that have already been closed/reopened -- so the admin
// picks the month directly. Closing locks writes for every report in that
// month and freezes a calculation run; its own purpose-built form here is
// the deliberate action, with no extra confirm step on top.
export function ClosePeriodModal({ open, onClose }: ClosePeriodModalProps) {
  const mutation = useClosePeriodMutation()
  const [month, setMonth] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  function resetForm() {
    setMonth('')
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
      const closed = await mutation.mutateAsync(month)
      toast.success(`${closed.month} is now closed.`)
      handleClose()
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
      title="Close a month"
      subtitle="Freezes that month's figures as a calculation run and blocks further writes to its daily reports. An administrator can reopen it with a reason."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="close-period-form" loading={mutation.isPending}>
            Close month
          </Button>
        </>
      }
    >
      <form
        id="close-period-form"
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

        <Field label="Month" htmlFor="close-period-month">
          <Input
            id="close-period-month"
            type="month"
            value={month}
            onChange={(event) => setMonth(event.target.value)}
            required
          />
        </Field>
      </form>
    </Modal>
  )
}
