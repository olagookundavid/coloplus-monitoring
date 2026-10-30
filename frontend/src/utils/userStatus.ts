import type { StatusTone } from '../components/ui'
import type { ManagedUser } from '../types'

export type ManagedUserStatus = 'Active' | 'Invited' | 'Suspended'

// The backend has no literal "suspended" state — just is_active and a
// derived invited flag (PasswordHash == ""). This is the one place that
// turns those two real fields into the three-way label the UI shows.
export function getManagedUserStatus(
  user: Pick<ManagedUser, 'isActive' | 'invited'>
): ManagedUserStatus {
  if (user.invited) return 'Invited'
  return user.isActive ? 'Active' : 'Suspended'
}

export function getManagedUserStatusTone(status: ManagedUserStatus): StatusTone {
  switch (status) {
    case 'Active':
      return 'success'
    case 'Invited':
      return 'warning'
    case 'Suspended':
      return 'neutral'
  }
}
