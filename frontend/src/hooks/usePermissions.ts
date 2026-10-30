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
    can: (permission: string) => permissions.includes(permission),
  }
}
