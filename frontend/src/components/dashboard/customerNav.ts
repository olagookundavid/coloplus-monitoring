import { DocumentTextIcon, Squares2X2Icon, TicketIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import { SiteMastIcon } from '../icons'
import type { NavItem } from './navTypes'

export const CUSTOMER_NAV: NavItem[] = [
  {
    label: 'Overview',
    to: ROUTES.HOME,
    icon: Squares2X2Icon,
    permission: 'customer:overview:view',
  },
  {
    label: 'Sites',
    to: ROUTES.CUSTOMER_SITES,
    icon: SiteMastIcon,
    permission: 'customer:sites:view',
  },
  {
    label: 'Support tickets',
    to: ROUTES.CUSTOMER_TICKETS,
    icon: TicketIcon,
    permission: 'customer:tickets:view',
    badge: 2,
  },
  {
    label: 'Reports',
    to: ROUTES.CUSTOMER_REPORTS,
    icon: DocumentTextIcon,
    permission: 'customer:reports:view',
  },
]
