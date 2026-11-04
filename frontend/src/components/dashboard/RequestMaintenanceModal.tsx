import { useState, type FormEvent } from 'react'

import { useAdminSites, useRequestMaintenanceMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import type { AdminSite, MaintenanceRules } from '../../types'
import {
  getFieldErrors,
  requestMaintenanceSchema,
  toast,
  type RequestMaintenanceFormValues,
} from '../../utils'
import { Button, Field, Input, Modal, Select, Textarea } from '../ui'

export interface RequestMaintenanceModalProps {
  open: boolean
  onClose: () => void
  rules?: MaintenanceRules
}

function nowAsDatetimeLocal(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const NO_SITES: AdminSite[] = []

function initialValues(): RequestMaintenanceFormValues {
  const now = nowAsDatetimeLocal()
  return { siteId: '', startsAt: now, endsAt: now, reason: '' }
}

export function RequestMaintenanceModal({ open, onClose, rules }: RequestMaintenanceModalProps) {
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const mutation = useRequestMaintenanceMutation()

  const [values, setValues] = useState<RequestMaintenanceFormValues>(initialValues)
  const [touched, setTouched] = useState<
    Partial<Record<keyof RequestMaintenanceFormValues, boolean>>
  >({})
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fieldErrors = getFieldErrors(requestMaintenanceSchema, values)

  function showError(field: keyof RequestMaintenanceFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange<K extends keyof RequestMaintenanceFormValues>(
    field: K,
    value: RequestMaintenanceFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof RequestMaintenanceFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  function resetForm() {
    setValues(initialValues())
    setTouched({})
    setSubmitted(false)
    setFormError(null)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await mutation.mutateAsync({
        siteId: values.siteId,
        startsAt: new Date(values.startsAt).toISOString(),
        endsAt: new Date(values.endsAt).toISOString(),
        reason: values.reason.trim(),
      })
      toast.success('Maintenance window requested.')
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
      title="Request maintenance"
      subtitle="Another person approves this before it's scheduled."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="request-maintenance-form" loading={mutation.isPending}>
            Request
          </Button>
        </>
      }
    >
      <form
        id="request-maintenance-form"
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

        <Field label="Site" htmlFor="maintenance-site" error={showError('siteId')}>
          <Select
            id="maintenance-site"
            value={values.siteId}
            invalid={Boolean(showError('siteId'))}
            onChange={(event) => handleChange('siteId', event.target.value)}
            onBlur={() => handleBlur('siteId')}
          >
            <option value="">Select a site</option>
            {sites.map((site) => (
              <option key={site.id} value={site.id}>
                {site.airtelCode} · {site.name}
              </option>
            ))}
          </Select>
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Starts at" htmlFor="maintenance-starts" error={showError('startsAt')}>
            <Input
              id="maintenance-starts"
              type="datetime-local"
              value={values.startsAt}
              invalid={Boolean(showError('startsAt'))}
              onChange={(event) => handleChange('startsAt', event.target.value)}
              onBlur={() => handleBlur('startsAt')}
            />
          </Field>
          <Field label="Ends at" htmlFor="maintenance-ends" error={showError('endsAt')}>
            <Input
              id="maintenance-ends"
              type="datetime-local"
              value={values.endsAt}
              invalid={Boolean(showError('endsAt'))}
              onChange={(event) => handleChange('endsAt', event.target.value)}
              onBlur={() => handleBlur('endsAt')}
            />
          </Field>
        </div>

        <Field
          label="Reason"
          htmlFor="maintenance-reason"
          error={showError('reason')}
          hint={
            rules
              ? `Needs at least ${rules.noticeHours}h notice, fits one day up to ${rules.maxHours}h, and a ${rules.monthlyCapHours}h/month cap per site.`
              : undefined
          }
        >
          <Textarea
            id="maintenance-reason"
            value={values.reason}
            invalid={Boolean(showError('reason'))}
            onChange={(event) => handleChange('reason', event.target.value)}
            onBlur={() => handleBlur('reason')}
          />
        </Field>
      </form>
    </Modal>
  )
}
