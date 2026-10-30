export const ROUTES = {
  LOGIN: '/login',
  FORGOT_PASSWORD: '/forgot-password',
  ACCEPT_INVITE: '/accept-invite',
  // ROUTES.HOME is also the customer portal's landing page (Overview) —
  // no separate CUSTOMER_OVERVIEW constant, to avoid two names drifting
  // apart for the same path.
  HOME: '/',
  CUSTOMER_SITES: '/sites',
  CUSTOMER_TICKETS: '/support-tickets',
  CUSTOMER_REPORTS: '/reports',
  SITE_DETAIL_PATTERN: '/sites/:id',
  TICKET_DETAIL_PATTERN: '/support-tickets/:id',
} as const

export function siteDetailPath(id: string): string {
  return `/sites/${id}`
}

export function ticketDetailPath(id: string): string {
  return `/support-tickets/${id}`
}
