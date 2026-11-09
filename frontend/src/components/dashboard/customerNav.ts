import {
  DocumentTextIcon,
  ExclamationTriangleIcon,
  ScaleIcon,
  Squares2X2Icon,
  TicketIcon,
} from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavGroup } from './navTypes'

// Permission strings are the real backend constants (from `permissions[]` on
// GET /auth/me) -- usePermissions().can() checks against that real array.
// viewer and reviewer share every permission gating these
// items (sites.read, published.read, disputes.read, acceptance.read), so
// there's no viewer/reviewer difference at the nav level -- only inside
// Disputes/Acceptance, where reviewer-only actions (raise, comment,
// inspect, accept, reject) are gated per-button. Support tickets stays
// unguarded (tickets.raise isn't a door-key permission, both roles hold it
// and so does nobody-else-relevant) -- same convention as before.
export const CUSTOMER_NAV_GROUPS: NavGroup[] = [
  {
    label: 'Workspace',
    items: [
      {
        label: 'Overview',
        to: ROUTES.HOME,
        icon: Squares2X2Icon,
        permission: 'sites.read',
      },
      {
        label: 'Sites',
        to: ROUTES.CUSTOMER_SITES,
        icon: SiteMastIcon,
        permission: 'sites.read',
      },
      {
        label: 'Incidents',
        to: ROUTES.CUSTOMER_INCIDENTS,
        icon: ExclamationTriangleIcon,
        permission: 'published.read',
      },
      {
        label: 'Disputes',
        to: ROUTES.CUSTOMER_DISPUTES,
        icon: ScaleIcon,
        permission: 'disputes.read',
      },
      {
        label: 'Acceptance',
        to: ROUTES.CUSTOMER_ACCEPTANCE,
        icon: SiteMastIcon,
        permission: 'acceptance.read',
      },
      {
        label: 'Support tickets',
        to: ROUTES.CUSTOMER_TICKETS,
        icon: TicketIcon,
      },
      {
        label: 'Reports',
        to: ROUTES.CUSTOMER_REPORTS,
        icon: DocumentTextIcon,
        permission: 'published.read',
      },
    ],
  },
]
