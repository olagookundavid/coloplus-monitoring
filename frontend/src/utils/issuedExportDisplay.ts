const KIND_LABELS: Record<string, string> = {
  daily_summaries: 'Daily summaries',
  incident_register: 'Incident register',
  monthly_performance: 'Monthly performance',
  weekly_exceptions: 'Weekly exceptions',
  commissioning_backlog: 'Commissioning backlog',
}

export function issuedExportKindLabel(kind: string): string {
  return KIND_LABELS[kind] ?? kind
}

// Not a users.manage-gated name lookup: supervisor sees this register too,
// scoped to their own exports only, and lacks users.manage -- comparing
// against the signed-in user's own id needs no extra fetch and works for
// every role that can reach this page.
export function issuedByLabel(issuedBy: string, currentUserId?: string): string {
  if (currentUserId && issuedBy === currentUserId) return 'You'
  return issuedBy.length > 8 ? `${issuedBy.slice(0, 8)}…` : issuedBy
}

export function describeIssuedExportParams(params: Record<string, unknown>): string {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  if (entries.length === 0) return '—'
  return entries.map(([key, value]) => `${key}: ${String(value)}`).join(' · ')
}
