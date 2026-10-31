import {
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  Cog6ToothIcon,
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
export const OPS_NAV_GROUPS: NavGroup[] = [
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
