import type { User } from '../../types'
import { STAFF_ROLE_LABELS, type StaffRole } from '../../types'
import { Modal } from '../ui'

export interface MyProfileModalProps {
  user: User
  open: boolean
  onClose: () => void
}

// Read-only: there's no self-service profile-edit endpoint (GET /auth/me is
// read-only), so this shows the real session, nothing more -- no fields that
// can't actually be saved anywhere.
export function MyProfileModal({ user, open, onClose }: MyProfileModalProps) {
  const roleLabel = STAFF_ROLE_LABELS[user.role as StaffRole] ?? user.role

  return (
    <Modal open={open} onClose={onClose} title="My profile" maxWidth="md">
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
    </Modal>
  )
}
