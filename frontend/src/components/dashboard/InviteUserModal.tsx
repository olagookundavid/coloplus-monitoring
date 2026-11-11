import { useState, type FormEvent } from 'react'

import { useAdminSites, useCreateUserMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import { STAFF_ROLES, STAFF_ROLE_LABELS, type AdminSite, type StaffRole } from '../../types'
import { getFieldErrors, inviteUserSchema, toast, type InviteUserFormValues } from '../../utils'
import { Button, Field, Input, Modal, Select } from '../ui'
import { SiteChecklist } from './SiteChecklist'

export interface InviteUserModalProps {
  open: boolean
  onClose: () => void
}

const INITIAL_VALUES: InviteUserFormValues = { email: '', name: '', role: 'reporter' }
const NO_SITES: AdminSite[] = []
const NO_SITE_IDS: string[] = []

export function InviteUserModal({ open, onClose }: InviteUserModalProps) {
  const mutation = useCreateUserMutation()
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES

  const [values, setValues] = useState<InviteUserFormValues>(INITIAL_VALUES)
  const [touched, setTouched] = useState<Partial<Record<keyof InviteUserFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [siteIds, setSiteIds] = useState<string[]>(NO_SITE_IDS)

  const fieldErrors = getFieldErrors(inviteUserSchema, values)

  function showError(field: keyof InviteUserFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange<K extends keyof InviteUserFormValues>(
    field: K,
    value: InviteUserFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof InviteUserFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  function resetForm() {
    setValues(INITIAL_VALUES)
    setTouched({})
    setSubmitted(false)
    setFormError(null)
    setSiteIds(NO_SITE_IDS)
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
        ...values,
        siteIds: values.role === 'admin' ? NO_SITE_IDS : siteIds,
      })
      toast.success(`Invited ${values.name}.`)
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
      title="Invite a user"
      subtitle="They'll get an email link to choose their own password. Choose the sites they can see below; you can change this later from their user page."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="invite-user-form" loading={mutation.isPending}>
            Send invite
          </Button>
        </>
      }
    >
      <form
        id="invite-user-form"
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

        <Field label="Full name" htmlFor="invite-user-name" error={showError('name')}>
          <Input
            id="invite-user-name"
            value={values.name}
            invalid={Boolean(showError('name'))}
            onChange={(event) => handleChange('name', event.target.value)}
            onBlur={() => handleBlur('name')}
          />
        </Field>

        <Field label="Email address" htmlFor="invite-user-email" error={showError('email')}>
          <Input
            id="invite-user-email"
            type="email"
            value={values.email}
            invalid={Boolean(showError('email'))}
            onChange={(event) => handleChange('email', event.target.value)}
            onBlur={() => handleBlur('email')}
          />
        </Field>

        <Field label="Role" htmlFor="invite-user-role">
          <Select
            id="invite-user-role"
            value={values.role}
            onChange={(event) => handleChange('role', event.target.value as StaffRole)}
          >
            {STAFF_ROLES.map((role) => (
              <option key={role} value={role}>
                {STAFF_ROLE_LABELS[role]}
              </option>
            ))}
          </Select>
        </Field>

        {values.role === 'admin' ? (
          <p className="rounded-lg bg-surface-subtle p-3 text-xs text-text-muted">
            Administrators can see every site, so no sites are chosen for them.
          </p>
        ) : (
          <div className="flex flex-col gap-1.5">
            <p className="text-sm font-medium text-text">Sites they can see</p>
            <SiteChecklist sites={sites} selected={siteIds} onChange={setSiteIds} />
          </div>
        )}
      </form>
    </Modal>
  )
}
