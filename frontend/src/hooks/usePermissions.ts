import { useAuth } from '../contexts'

/**
 * Minimal stub (architecture standard §4 names this as the permission-gating
 * mechanism). No permission/role claims exist in the token payload yet
 * (pkg-common Payload = {UserID, IssuedAt, ExpiredAt} only), so `can()` is
 * placeholder-true for any authenticated user until the backend ships real
 * claims. The backend remains the authority regardless (frontend/CLAUDE.md
 * §14) — this hook is UI convenience, not the security boundary.
 */
export function usePermissions() {
  const { user } = useAuth()
  return {
    isAuthenticated: Boolean(user),
    can: (_permission: string) => Boolean(user),
  }
}
