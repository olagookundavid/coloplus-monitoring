import { Squares2X2Icon, UserGroupIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavItem } from './navTypes'

// Every item gated on users.manage: the one real permission admin holds
// that covers all three pages built so far. Approvals/Formulas/Audit
// log/Settings aren't here yet — no page exists for them, and a nav link
// to nothing is a dead link, not a feature.
export const OPS_NAV: NavItem[] = [
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
]
