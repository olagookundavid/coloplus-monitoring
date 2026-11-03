import { ArrowPathIcon } from '@heroicons/react/24/outline'

import { useDigestPreference, useUpdateDigestPreferenceMutation } from '../../hooks'
import type { User } from '../../types'
import { STAFF_ROLE_LABELS, type StaffRole } from '../../types'
import { toast } from '../../utils'
import { Checkbox, Modal } from '../ui'

export interface MyProfileModalProps {
  user: User
  open: boolean
  onClose: () => void
}

// The session facts (name/email/role) stay read-only -- there's no
// self-service profile-edit endpoint for them. The weekly summary email
// below it is genuinely self-service (GET/PUT /digest/preference).
export function MyProfileModal({ user, open, onClose }: MyProfileModalProps) {
  const roleLabel = STAFF_ROLE_LABELS[user.role as StaffRole] ?? user.role

  const digestQuery = useDigestPreference(open)
  const updateDigestMutation = useUpdateDigestPreferenceMutation()

  async function handleDigestToggle(weekly: boolean) {
    try {
      await updateDigestMutation.mutateAsync(weekly)
      toast.success(weekly ? 'Weekly summary email turned on.' : 'Weekly summary email turned off.')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Something went wrong. Try again.')
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="My profile" maxWidth="md">
      <div className="flex flex-col gap-4">
        <dl className="divide-y divide-border rounded-lg border border-border">
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <dt className="text-sm text-text-muted">Name</dt>
            <dd className="text-sm font-medium text-text">{user.name}</dd>
          </div>
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <dt className="text-sm text-text-muted">Email</dt>
            <dd className="text-sm font-medium text-text">{user.email}</dd>
          </div>
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <dt className="text-sm text-text-muted">Role</dt>
            <dd className="text-sm font-medium text-text">{roleLabel}</dd>
          </div>
        </dl>

        <div className="border-t border-border pt-4">
          {digestQuery.isLoading ? (
            <p className="flex items-center gap-2 text-sm text-text-muted">
              <ArrowPathIcon className="h-4 w-4 animate-spin" aria-hidden="true" />
              Loading preference…
            </p>
          ) : (
            <Checkbox
              label="Weekly summary email"
              hint="A Monday-morning summary of your own sites: outages, open incidents, low fuel where visible, and sites awaiting a report."
              checked={digestQuery.data?.weekly ?? true}
              disabled={updateDigestMutation.isPending}
              onChange={(event) => void handleDigestToggle(event.target.checked)}
            />
          )}
        </div>
      </div>
    </Modal>
  )
}
