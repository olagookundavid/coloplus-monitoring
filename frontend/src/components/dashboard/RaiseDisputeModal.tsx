import { useState, type FormEvent } from 'react'
import { ScaleIcon } from '@heroicons/react/24/outline'

import { useAdminSites, useRaiseDisputeMutation } from '../../hooks'
import type { DisputeCategory } from '../../types'
import { toast } from '../../utils'
import { Button, Field, Input, Select, SidePanel, Textarea } from '../ui'

export interface RaiseDisputeModalProps {
  open: boolean
  onClose: () => void
  /** Pre-selects the site when opened from that site's own detail panel. */
  initialSiteId?: string
}

const DISPUTE_CATEGORIES: { value: DisputeCategory; label: string }[] = [
  { value: 'incorrect_outage', label: 'Outage shown that did not happen' },
  { value: 'missing_outage', label: 'An outage is missing' },
  { value: 'incorrect_served', label: 'Served time looks wrong' },
  { value: 'wrong_duration', label: 'Duration looks wrong' },
  { value: 'wrong_metric', label: 'A calculated figure looks wrong' },
  { value: 'missing_report', label: 'A report is missing' },
  { value: 'other', label: 'Other' },
]

interface FormValues {
  siteId: string
  category: DisputeCategory
  title: string
  description: string
  periodFrom: string
  periodTo: string
}

const INITIAL_VALUES: FormValues = {
  siteId: '',
  category: 'incorrect_outage',
  title: '',
  description: '',
  periodFrom: '',
  periodTo: '',
}

export function RaiseDisputeModal({ open, onClose, initialSiteId }: RaiseDisputeModalProps) {
  const [values, setValues] = useState<FormValues>(INITIAL_VALUES)
  const [submitted, setSubmitted] = useState(false)

  const sitesQuery = useAdminSites()
  const mutation = useRaiseDisputeMutation()

  // Same re-prefill-on-open pattern as CreateTicketModal.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setValues({ ...INITIAL_VALUES, siteId: initialSiteId ?? '' })
  }

  function handleChange<K extends keyof FormValues>(field: K, value: FormValues[K]) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function resetForm() {
    setValues(INITIAL_VALUES)
    setSubmitted(false)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  const canSubmit = Boolean(values.siteId && values.title.trim() && values.description.trim())

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    if (!canSubmit) return

    try {
      await mutation.mutateAsync({
        siteId: values.siteId,
        category: values.category,
        title: values.title.trim(),
        description: values.description.trim(),
        periodFrom: values.periodFrom || undefined,
        periodTo: values.periodTo || undefined,
      })
      toast.success('Dispute raised.')
      handleClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not raise this dispute.')
    }
  }

  return (
    <SidePanel
      open={open}
      onClose={handleClose}
      eyebrow="Dispute a published figure"
      title="Raise a dispute"
      subtitle="Disputes apply to an approved report, site or period — only Coloplus can correct the underlying record."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="raise-dispute-form"
            icon={ScaleIcon}
            loading={mutation.isPending}
            disabled={submitted && !canSubmit}
          >
            Raise dispute
          </Button>
        </>
      }
    >
      <form
        id="raise-dispute-form"
        noValidate
        onSubmit={handleSubmit}
        className="flex flex-col gap-4"
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field
            label="Site"
            htmlFor="dispute-site"
            error={submitted && !values.siteId ? 'Select a site.' : undefined}
          >
            <Select
              id="dispute-site"
              value={values.siteId}
              invalid={submitted && !values.siteId}
              onChange={(event) => handleChange('siteId', event.target.value)}
            >
              <option value="">Select a site</option>
              {(sitesQuery.data ?? []).map((site) => (
                <option key={site.id} value={site.id}>
                  {site.name} ({site.customerSiteCode})
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Category" htmlFor="dispute-category">
            <Select
              id="dispute-category"
              value={values.category}
              onChange={(event) => handleChange('category', event.target.value as DisputeCategory)}
            >
              {DISPUTE_CATEGORIES.map((category) => (
                <option key={category.value} value={category.value}>
                  {category.label}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field
          label="Title"
          htmlFor="dispute-title"
          error={submitted && !values.title.trim() ? 'Give the dispute a short title.' : undefined}
        >
          <Input
            id="dispute-title"
            placeholder="e.g. Outage shown on a day power was on"
            value={values.title}
            invalid={submitted && !values.title.trim()}
            onChange={(event) => handleChange('title', event.target.value)}
          />
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Period from (optional)" htmlFor="dispute-from">
            <Input
              id="dispute-from"
              type="date"
              value={values.periodFrom}
              onChange={(event) => handleChange('periodFrom', event.target.value)}
            />
          </Field>
          <Field label="Period to (optional)" htmlFor="dispute-to">
            <Input
              id="dispute-to"
              type="date"
              value={values.periodTo}
              onChange={(event) => handleChange('periodTo', event.target.value)}
            />
          </Field>
        </div>

        <Field
          label="What looks wrong?"
          htmlFor="dispute-description"
          error={
            submitted && !values.description.trim() ? 'Describe what looks wrong.' : undefined
          }
        >
          <Textarea
            id="dispute-description"
            placeholder="Tell us what you observed and why it doesn't match the published figure."
            value={values.description}
            invalid={submitted && !values.description.trim()}
            onChange={(event) => handleChange('description', event.target.value)}
          />
        </Field>
      </form>
    </SidePanel>
  )
}
