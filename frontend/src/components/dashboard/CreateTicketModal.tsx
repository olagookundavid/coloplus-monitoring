import { useState, type FormEvent } from 'react'
import { TicketIcon } from '@heroicons/react/24/outline'

import { useAdminSites, useCreateTicketMutation } from '../../hooks'
import type { TicketCategory, TicketPriority } from '../../types'
import { createTicketSchema, getFieldErrors, toast, type CreateTicketFormValues } from '../../utils'
import { Button, Field, Input, Select, SidePanel, Textarea } from '../ui'

export interface CreateTicketModalProps {
  open: boolean
  onClose: () => void
  /** Pre-selects the site when opened from that site's own detail panel. */
  initialSiteId?: string
}

const TICKET_CATEGORIES: { value: TicketCategory; label: string }[] = [
  { value: 'power_outage', label: 'Power outage' },
  { value: 'low_fuel', label: 'Low fuel / diesel' },
  { value: 'generator_fault', label: 'Generator fault' },
  { value: 'equipment_fault', label: 'Equipment fault' },
  { value: 'access_restriction', label: 'Access restriction' },
  { value: 'other', label: 'Other' },
]

const TICKET_PRIORITIES: { value: TicketPriority; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'critical', label: 'Critical' },
]

const INITIAL_VALUES: CreateTicketFormValues = {
  siteId: '',
  category: 'power_outage',
  subject: '',
  priority: 'medium',
  contactEmail: '',
  description: '',
}

export function CreateTicketModal({ open, onClose, initialSiteId }: CreateTicketModalProps) {
  const [values, setValues] = useState<CreateTicketFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof CreateTicketFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)

  const sitesQuery = useAdminSites()
  const mutation = useCreateTicketMutation()

  // Re-prefill every time the modal opens, not just on mount — the same
  // modal instance stays alive across opens (owned by CustomerLayout), and
  // a site's own "Create ticket" action should always pre-select that site.
  // Adjusted synchronously during render (React's "adjusting state when a
  // prop changes" pattern) rather than in an effect, per this project's
  // React Compiler lint rules (see SidePanel.tsx for the same pattern).
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setValues({ ...INITIAL_VALUES, siteId: initialSiteId ?? '' })
  }

  const fieldErrors = getFieldErrors(createTicketSchema, values)

  function showError(field: keyof CreateTicketFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange<K extends keyof CreateTicketFormValues>(
    field: K,
    value: CreateTicketFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof CreateTicketFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  function resetForm() {
    setValues(INITIAL_VALUES)
    setTouched({})
    setSubmitted(false)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      await mutation.mutateAsync({
        siteId: values.siteId,
        category: values.category,
        priority: values.priority,
        subject: values.subject,
        description: values.description,
        contactEmail: values.contactEmail || undefined,
      })
      toast.success('Ticket created.')
      handleClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Something went wrong. Try again.')
    }
  }

  return (
    <SidePanel
      open={open}
      onClose={handleClose}
      eyebrow="Coloplus support"
      title="How can we help?"
      subtitle="Create a request linked to your site and track it here."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="create-ticket-form"
            icon={TicketIcon}
            loading={mutation.isPending}
          >
            Create ticket
          </Button>
        </>
      }
    >
      <form
        id="create-ticket-form"
        noValidate
        onSubmit={handleSubmit}
        className="flex flex-col gap-4"
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Affected site" htmlFor="ticket-site" error={showError('siteId')}>
            <Select
              id="ticket-site"
              value={values.siteId}
              invalid={Boolean(showError('siteId'))}
              onChange={(event) => handleChange('siteId', event.target.value)}
              onBlur={() => handleBlur('siteId')}
            >
              <option value="">Select a site</option>
              {(sitesQuery.data ?? []).map((site) => (
                <option key={site.id} value={site.id}>
                  {site.name} ({site.customerSiteCode})
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Category" htmlFor="ticket-category">
            <Select
              id="ticket-category"
              value={values.category}
              onChange={(event) => handleChange('category', event.target.value as TicketCategory)}
            >
              {TICKET_CATEGORIES.map((category) => (
                <option key={category.value} value={category.value}>
                  {category.label}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field label="Subject" htmlFor="ticket-subject" error={showError('subject')}>
          <Input
            id="ticket-subject"
            placeholder="Briefly describe the issue"
            value={values.subject}
            invalid={Boolean(showError('subject'))}
            onChange={(event) => handleChange('subject', event.target.value)}
            onBlur={() => handleBlur('subject')}
          />
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Priority" htmlFor="ticket-priority">
            <Select
              id="ticket-priority"
              value={values.priority}
              onChange={(event) => handleChange('priority', event.target.value as TicketPriority)}
            >
              {TICKET_PRIORITIES.map((priority) => (
                <option key={priority.value} value={priority.value}>
                  {priority.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field
            label="Contact email"
            htmlFor="ticket-email"
            error={showError('contactEmail')}
            hint="Leave blank to use your account's email."
          >
            <Input
              id="ticket-email"
              type="email"
              placeholder="name@company.com"
              value={values.contactEmail}
              invalid={Boolean(showError('contactEmail'))}
              onChange={(event) => handleChange('contactEmail', event.target.value)}
              onBlur={() => handleBlur('contactEmail')}
            />
          </Field>
        </div>

        <Field label="What happened?" htmlFor="ticket-description" error={showError('description')}>
          <Textarea
            id="ticket-description"
            placeholder="Tell us the impact, when it started and any relevant details."
            value={values.description}
            invalid={Boolean(showError('description'))}
            onChange={(event) => handleChange('description', event.target.value)}
            onBlur={() => handleBlur('description')}
          />
        </Field>
      </form>
    </SidePanel>
  )
}
