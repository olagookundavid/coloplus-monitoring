import {
  ArrowUturnLeftIcon,
  CalculatorIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  Cog6ToothIcon,
  DocumentTextIcon,
  EnvelopeIcon,
  ExclamationTriangleIcon,
  LockClosedIcon,
  MapIcon,
  Squares2X2Icon,
  UserGroupIcon,
  WrenchScrewdriverIcon,
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
//
// Maintenance has a copy in all three "Workspace" groups below, same as
// Approvals/Reports to review -- but each copy is gated on that group's own
// identity permission (users.manage / reports.review / reports.create), NOT
// on maintenance.request itself. The Sidebar filters each group's items by
// permission independently, so gating every copy on maintenance.request
// (which all three roles hold) would make every group "survive" filtering
// for every role -- an admin would see it three times, once per group,
// including two orphaned "Workspace" headings with nothing else in them.
// Gating each copy on its own group's mutually exclusive identity
// permission means exactly one copy is ever visible, and it always lands
// inside that role's own already-surviving group. maintenance.request still
// guards the route itself (App.tsx) -- this only decides nav placement.
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
        // Gated on users.manage, not exports.create: the backend's real
        // check for /calculation-runs is exports.create AND Actor.AllSites
        // AND NOT published.read, and only admin currently holds AllSites.
        // exports.create is also held by supervisor and both Airtel roles,
        // who would get a 403 if this nav item used that permission name.
        {
          label: 'Calculation runs',
          to: ROUTES.OPS_CALCULATION_RUNS,
          icon: CalculatorIcon,
          permission: 'users.manage',
        },
        {
          label: 'Periods',
          to: ROUTES.OPS_PERIODS,
          icon: LockClosedIcon,
          permission: 'periods.manage',
        },
        {
          label: 'Maintenance',
          to: ROUTES.OPS_MAINTENANCE,
          icon: WrenchScrewdriverIcon,
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
        {
          label: 'Maintenance',
          to: ROUTES.OPS_MAINTENANCE,
          icon: WrenchScrewdriverIcon,
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
        {
          label: 'Maintenance',
          to: ROUTES.OPS_MAINTENANCE,
          icon: WrenchScrewdriverIcon,
          permission: 'reports.create',
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
