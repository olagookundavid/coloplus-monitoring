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

import { nameOrId } from './names'

export function issuedByLabel(
  issuedBy: string,
  issuedByName: string | undefined,
  currentUserId?: string
): string {
  if (currentUserId && issuedBy === currentUserId) return 'You'
  return nameOrId(issuedByName, issuedBy)
}

export function describeIssuedExportParams(
  params: Record<string, unknown>,
  siteName?: string
): string {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  if (entries.length === 0) return '—'
  return entries
    .map(([key, value]) =>
      key === 'site_id' ? `site: ${nameOrId(siteName, String(value))}` : `${key}: ${String(value)}`
    )
    .join(' · ')
}
