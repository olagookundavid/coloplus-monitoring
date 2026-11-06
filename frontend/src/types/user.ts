// The real, backend-known roles (backend/internal/domain/user.go) — never
// invent a role here that has no permission set behind it.
export type StaffRole =
  'admin' | 'reporter' | 'supervisor' | 'airtel_viewer' | 'airtel_reviewer' | 'auditor'

export const STAFF_ROLE_LABELS: Record<StaffRole, string> = {
  admin: 'Coloplus administrator',
  reporter: 'Field officer',
  supervisor: 'Operations supervisor',
  airtel_viewer: 'Airtel viewer',
  airtel_reviewer: 'Airtel reviewer',
  auditor: 'Auditor',
}

export const STAFF_ROLES: StaffRole[] = [
  'admin',
  'reporter',
  'supervisor',
  'airtel_viewer',
  'airtel_reviewer',
  'auditor',
]

import type { SiteRef } from './adminSite'

// The admin's view of *another* account — distinct from types/auth.ts's
// `User`, which is the signed-in session's own shape from /auth/me.
// `siteIds` is always present (both the list and single-user fetches return
// it, batch-loaded server-side); still optional here since an admin's own
// row carries none. `sites` names the same assignment, same order.
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
  sites?: SiteRef[]
}
