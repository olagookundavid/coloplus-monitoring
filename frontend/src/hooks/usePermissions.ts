import { useAuth } from '../contexts'

/**
 * Thin read of the real permission set from `GET /auth/me`
 * (architecture standard §4). The backend remains the authority
 * (frontend/CLAUDE.md §14) — this hook is UI convenience, not the
 * security boundary.
 */
export function usePermissions() {
  const { user, permissions, allSites, siteIds } = useAuth()
  return {
    isAuthenticated: Boolean(user),
    role: user?.role,
    allSites,
    siteIds,
    // A list is "any of" -- e.g. an Approvals route that an admin (view
    // only, no reports.review) and a supervisor (can act) both reach.
    can: (permission: string | string[]) =>
      Array.isArray(permission)
        ? permission.some((p) => permissions.includes(p))
        : permissions.includes(permission),
  }
}
