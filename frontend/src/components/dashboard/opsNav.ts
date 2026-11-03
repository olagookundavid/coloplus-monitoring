import {
  ArrowUturnLeftIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  Cog6ToothIcon,
  DocumentTextIcon,
  EnvelopeIcon,
  ExclamationTriangleIcon,
  MapIcon,
  Squares2X2Icon,
  UserGroupIcon,
} from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavGroup } from './navTypes'

// Overview/Users/Site assignments/Settings/Approvals are admin-only
// (users.manage), Audit log for anyone holding audit.read, and Email log for
// anyone holding notifications.read -- each gated on the real permission
// that controls its data, not users.manage, via its own RequireRole block in
// App.tsx. Formulas isn't here yet — no page exists for it, and a nav link
// to nothing is a dead link, not a feature.
//
// The third "Workspace" group below is the supervisor's own portal
// (reports.review) -- same group label as admin's and the reporter's above
// and below (frontend/CLAUDE.md's per-role sidebar convention), distinct
// permission. Its "Reports to review" item points at the same OPS_APPROVALS
// route as admin's "Approvals" above, just under a name that fits what a
// supervisor is actually there to do. Both items existing side by side
// would be a duplicate row for nobody (admin lacks reports.review,
// supervisor lacks users.manage, per Decision 025), so each role sees
// exactly one.
//
// buildOpsNavGroups takes a live Corrections badge count (how many of the
// reporter's own reports are currently returned) -- a zero-arg call renders
// the group without one, for a caller with nothing to count yet.
export function buildOpsNavGroups(correctionsBadge = 0): NavGroup[] {
  return [
    {
      label: 'Workspace',
      items: [
        {
          label: 'Overview',
          to: ROUTES.OPS_OVERVIEW,
          icon: Squares2X2Icon,
          permission: 'users.manage',
        },
        {
          label: 'Users',
          to: ROUTES.OPS_USERS,
          icon: UserGroupIcon,
          permission: 'users.manage',
        },
        {
          label: 'Sites',
          to: ROUTES.OPS_SITES,
          icon: MapIcon,
          permission: 'users.manage',
        },
        {
          label: 'Site assignments',
          to: ROUTES.OPS_SITE_ASSIGNMENTS,
          icon: SiteMastIcon,
          permission: 'users.manage',
        },
        {
          label: 'Approvals',
          to: ROUTES.OPS_APPROVALS,
          icon: ClipboardDocumentCheckIcon,
          permission: 'users.manage',
        },
        {
          label: 'Audit log',
          to: ROUTES.OPS_AUDIT_LOG,
          icon: ClipboardDocumentListIcon,
          permission: 'audit.read',
        },
        {
          label: 'Email log',
          to: ROUTES.OPS_EMAIL_LOG,
          icon: EnvelopeIcon,
          permission: 'notifications.read',
        },
      ],
    },
    {
      label: 'Workspace',
      items: [
        {
          label: 'Overview',
          to: ROUTES.OPS_SUPERVISOR_OVERVIEW,
          icon: Squares2X2Icon,
          permission: 'reports.review',
        },
        {
          label: 'Sites',
          to: ROUTES.OPS_SUPERVISOR_SITES,
          icon: MapIcon,
          permission: 'reports.review',
        },
        {
          label: 'Incidents',
          to: ROUTES.OPS_SUPERVISOR_INCIDENTS,
          icon: ExclamationTriangleIcon,
          permission: 'reports.review',
        },
        {
          label: 'Reports to review',
          to: ROUTES.OPS_APPROVALS,
          icon: ClipboardDocumentCheckIcon,
          permission: 'reports.review',
        },
      ],
    },
    {
      label: 'Workspace',
      items: [
        {
          label: 'Overview',
          to: ROUTES.OPS_REPORTER_OVERVIEW,
          icon: Squares2X2Icon,
          permission: 'reports.create',
        },
        {
          label: 'My sites',
          to: ROUTES.OPS_MY_SITES,
          icon: SiteMastIcon,
          permission: 'reports.create',
        },
        {
          label: 'Daily reports',
          to: ROUTES.OPS_DAILY_REPORTS,
          icon: DocumentTextIcon,
          permission: 'reports.create',
        },
        {
          label: 'Incidents',
          to: ROUTES.OPS_INCIDENTS,
          icon: ExclamationTriangleIcon,
          permission: 'reports.create',
        },
        {
          label: 'Corrections',
          to: ROUTES.OPS_CORRECTIONS,
          icon: ArrowUturnLeftIcon,
          permission: 'reports.create',
          badge: correctionsBadge || undefined,
        },
      ],
    },
    {
      label: 'System',
      items: [
        {
          label: 'Settings',
          to: ROUTES.OPS_SETTINGS,
          icon: Cog6ToothIcon,
          permission: 'users.manage',
        },
      ],
    },
  ]
}

export const OPS_NAV_GROUPS: NavGroup[] = buildOpsNavGroups()
