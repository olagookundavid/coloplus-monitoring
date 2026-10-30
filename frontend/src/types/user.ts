// The real, backend-known roles (backend/internal/domain/user.go) — never
// invent a role here that has no permission set behind it.
export type StaffRole = 'admin' | 'reporter' | 'supervisor' | 'airtel_viewer' | 'airtel_reviewer'

export const STAFF_ROLE_LABELS: Record<StaffRole, string> = {
  admin: 'Coloplus administrator',
  reporter: 'Field officer',
  supervisor: 'Operations supervisor',
  airtel_viewer: 'Airtel viewer',
  airtel_reviewer: 'Airtel reviewer',
}

export const STAFF_ROLES: StaffRole[] = [
  'admin',
  'reporter',
  'supervisor',
  'airtel_viewer',
  'airtel_reviewer',
]

// The admin's view of *another* account — distinct from types/auth.ts's
// `User`, which is the signed-in session's own shape from /auth/me.
// `siteIds` is only present when fetched via fetchUser(id); the list
// endpoint doesn't return it.
export interface ManagedUser {
  id: string
  email: string
  name: string
  role: StaffRole
  isActive: boolean
  invited: boolean
  mfaEnabled: boolean
  lastLoginAt?: string
  siteIds?: string[]
}
