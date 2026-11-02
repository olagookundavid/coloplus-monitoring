import { DocumentTextIcon, Squares2X2Icon, TicketIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavGroup } from './navTypes'

// Permission strings are the real backend constants (from `permissions[]` on
// GET /auth/me), not invented ones — usePermissions().can() checks against
// that real array, so a placeholder string here would just hide the item
// from every real user. Support tickets has no backend permission concept at
// all (the feature isn't built server-side yet), so it stays unguarded
// (visible to any signed-in customer-portal user) rather than inventing one.
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
        label: 'Support tickets',
        to: ROUTES.CUSTOMER_TICKETS,
        icon: TicketIcon,
        badge: 2,
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
