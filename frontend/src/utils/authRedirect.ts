import { ROUTES } from '../constants'

// Permission-based, not role-based (frontend/CLAUDE.md §14) -- and checked
// against every permission that actually unlocks an /ops/* page, not just
// admin's, so a supervisor (reports.review, no users.manage) lands
// somewhere real instead of the customer portal.
export function landingRouteFor(permissions: string[]): string {
  if (permissions.includes('users.manage')) return ROUTES.OPS_OVERVIEW
  if (permissions.includes('reports.review')) return ROUTES.OPS_APPROVALS
  return ROUTES.HOME
}
