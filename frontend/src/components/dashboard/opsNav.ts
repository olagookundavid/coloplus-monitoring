import {
  ArrowUturnLeftIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  Cog6ToothIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  Squares2X2Icon,
  UserGroupIcon,
} from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavGroup } from './navTypes'

// Overview/Users/Site assignments/Settings are admin-only (users.manage).
// Approvals also opens for a supervisor (reports.review, no users.manage),
// and Audit log for anyone holding audit.read -- each gated on the real
// permission that controls its data, not users.manage, via its own
// RequireRole block in App.tsx. Formulas isn't here yet — no page exists
// for it, and a nav link to nothing is a dead link, not a feature.
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
          label: 'Site assignments',
          to: ROUTES.OPS_SITE_ASSIGNMENTS,
          icon: SiteMastIcon,
          permission: 'users.manage',
        },
        {
          label: 'Approvals',
          to: ROUTES.OPS_APPROVALS,
          icon: ClipboardDocumentCheckIcon,
          permission: ['users.manage', 'reports.review'],
        },
        {
          label: 'Audit log',
          to: ROUTES.OPS_AUDIT_LOG,
          icon: ClipboardDocumentListIcon,
          permission: 'audit.read',
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
