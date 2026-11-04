import { ROUTES } from '../constants'

// Permission-based, not role-based (frontend/CLAUDE.md §14) -- and checked
// against every permission that actually unlocks an /ops/* page, not just
// admin's, so a supervisor (reports.review, no users.manage) lands
// somewhere real instead of the customer portal.
export function landingRouteFor(permissions: string[]): string {
  if (permissions.includes('users.manage')) return ROUTES.OPS_OVERVIEW
  if (permissions.includes('reports.review')) return ROUTES.OPS_SUPERVISOR_OVERVIEW
  if (permissions.includes('reports.create')) return ROUTES.OPS_REPORTER_OVERVIEW
  // Checked last: admin also holds audit.read, but always resolves via
  // users.manage above first, so this is reached only by a real auditor.
  // Without this branch, auditor fell through to ROUTES.HOME (published.read-
  // gated), got denied there too, and landingRouteFor was called again with
  // the same permissions -- an infinite redirect loop on every auditor login.
  if (permissions.includes('audit.read')) return ROUTES.OPS_AUDITOR_OVERVIEW
  return ROUTES.HOME
}
