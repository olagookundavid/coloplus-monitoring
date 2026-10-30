import { useState, type FormEvent } from 'react'
import { TicketIcon } from '@heroicons/react/24/outline'

import { useCreateTicketMutation, useSitePerformance } from '../../hooks'
import type { TicketCategory, TicketPriority } from '../../types'
import { createTicketSchema, getFieldErrors, toast, type CreateTicketFormValues } from '../../utils'
import { Button, Field, Input, Modal, Select, Textarea } from '../ui'

export interface CreateTicketModalProps {
  open: boolean
  onClose: () => void
}

const TICKET_CATEGORIES: TicketCategory[] = [
  'Power outage',
  'Low fuel / diesel',
  'Generator fault',
  'Equipment fault',
  'Access restriction',
  'Other',
]

const TICKET_PRIORITIES: TicketPriority[] = ['Low', 'Medium', 'High', 'Critical']

const INITIAL_VALUES: CreateTicketFormValues = {
  siteCode: '',
  category: 'Power outage',
  subject: '',
  priority: 'Medium',
  contactEmail: '',
  description: '',
}

export function CreateTicketModal({ open, onClose }: CreateTicketModalProps) {
  const [values, setValues] = useState<CreateTicketFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof CreateTicketFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [attachmentNames, setAttachmentNames] = useState<string[]>([])

  const sitesQuery = useSitePerformance({})
  const mutation = useCreateTicketMutation()

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
    setAttachmentNames([])
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
      await mutation.mutateAsync({ ...values, attachmentNames })
      toast.success('Ticket created.')
      handleClose()
    } catch {
      toast.error('Something went wrong. Try again.')
    }
  }

  return (
    <Modal
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
            Create demo ticket
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
          <Field label="Affected site" htmlFor="ticket-site" error={showError('siteCode')}>
            <Select
              id="ticket-site"
              value={values.siteCode}
              invalid={Boolean(showError('siteCode'))}
              onChange={(event) => handleChange('siteCode', event.target.value)}
              onBlur={() => handleBlur('siteCode')}
            >
              <option value="">Select a site</option>
              {(sitesQuery.data ?? []).map((site) => (
                <option key={site.id} value={site.code}>
                  {site.name} ({site.code})
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
                <option key={category} value={category}>
                  {category}
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
                <option key={priority} value={priority}>
                  {priority}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Contact email" htmlFor="ticket-email" error={showError('contactEmail')}>
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

        {/* Native, unstyled file input — the mockup itself shows the
            browser's default file chooser, not a custom one, and no
            primitive exists for file inputs (frontend/CLAUDE.md §7 doesn't
            cover this control type). Filenames only, matching the hint
            below: nothing is actually uploaded. */}
        <Field
          label="Supporting files"
          htmlFor="ticket-files"
          hint="Prototype stores filenames only. No file content is uploaded."
        >
          <input
            id="ticket-files"
            type="file"
            multiple
            className="block w-full cursor-pointer text-sm text-text file:mr-3 file:cursor-pointer file:rounded-lg file:border file:border-border file:bg-surface-subtle file:px-3 file:py-2 file:text-sm file:font-medium file:text-text"
            onChange={(event) =>
              setAttachmentNames(Array.from(event.target.files ?? []).map((file) => file.name))
            }
          />
        </Field>

        <p className="rounded-lg bg-surface-subtle p-3 text-xs text-text-muted">
          Interactive demo: submitting creates a local sample ticket. A production backend will
          route it to the Coloplus support team.
        </p>
      </form>
    </Modal>
  )
}
