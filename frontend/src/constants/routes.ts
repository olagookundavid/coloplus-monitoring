export const ROUTES = {
  LOGIN: '/login',
  FORGOT_PASSWORD: '/forgot-password',
  RESET_PASSWORD: '/reset-password',
  ACCEPT_INVITE: '/accept-invite',
  MFA_SETUP: '/mfa-setup',
  // ROUTES.HOME is also the customer portal's landing page (Overview) —
  // no separate CUSTOMER_OVERVIEW constant, to avoid two names drifting
  // apart for the same path.
  HOME: '/',
  CUSTOMER_SITES: '/sites',
  CUSTOMER_TICKETS: '/support-tickets',
  CUSTOMER_REPORTS: '/reports',
  // "/incidents" is already OPS_INCIDENTS (the reporter's own page) below --
  // this one is Airtel's published-incident register, a different portal.
  CUSTOMER_INCIDENTS: '/incident-register',
  CUSTOMER_DISPUTES: '/disputes',
  CUSTOMER_ACCEPTANCE: '/acceptance',
  SITE_DETAIL_PATTERN: '/sites/:id',
  TICKET_DETAIL_PATTERN: '/support-tickets/:id',
  INCIDENT_DETAIL_PATTERN: '/incident-register/:id',
  // These names stay OPS_* (the Coloplus-staff portal they belong to,
  // pages/ops/) even though their URLs carry no "/ops" prefix -- the portal
  // split is an internal/file-structure concern, not something the URL bar
  // should reveal. Access is controlled by RequireRole's permission checks
  // in App.tsx, not by the shape of the path.
  OPS_OVERVIEW: '/overview',
  OPS_USERS: '/users',
  OPS_SITES: '/site-directory',
  OPS_SITE_ASSIGNMENTS: '/site-assignments',
  OPS_APPROVALS: '/approvals',
  OPS_AUDIT_LOG: '/audit-log',
  OPS_EMAIL_LOG: '/email-log',
  OPS_SETTINGS: '/settings',
  OPS_PERIODS: '/periods',
  OPS_MAINTENANCE: '/maintenance',
  OPS_ISSUED_EXPORTS: '/issued-exports',
  OPS_CALCULATION_RUNS: '/calculation-runs',
  CALCULATION_RUN_DETAIL_PATTERN: '/calculation-runs/:id',
  OPS_REPORTER_OVERVIEW: '/reporting',
  OPS_MY_SITES: '/my-sites',
  OPS_DAILY_REPORTS: '/daily-reports',
  OPS_INCIDENTS: '/incidents',
  OPS_CORRECTIONS: '/corrections',
  // The supervisor portal. OPS_OVERVIEW/OPS_INCIDENTS above are already taken
  // by admin/reporter, so these get their own slugs even though they're
  // conceptually "overview"/"incidents" for a third ops role.
  OPS_SUPERVISOR_OVERVIEW: '/supervision',
  OPS_SUPERVISOR_SITES: '/site-oversight',
  OPS_SUPERVISOR_INCIDENTS: '/incident-management',
} as const

export function siteDetailPath(id: string): string {
  return `/sites/${id}`
}

export function ticketDetailPath(id: string): string {
  return `/support-tickets/${id}`
}

export function incidentDetailPath(id: string): string {
  return `/incident-register/${id}`
}

export function calculationRunDetailPath(id: string): string {
  return `/calculation-runs/${id}`
}
