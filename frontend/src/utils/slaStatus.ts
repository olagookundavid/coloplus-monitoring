import type { StatusTone } from '../components/ui'
import type { SLA } from '../types'

export function slaStatusLabel(status: SLA['status']): { label: string; tone: StatusTone } {
  switch (status) {
    case 'met':
      return { label: 'SLA met', tone: 'success' }
    case 'at_risk':
      return { label: 'SLA at risk', tone: 'warning' }
    case 'missed':
      return { label: 'SLA missed', tone: 'danger' }
  }
}

// Always shown beside the status above, never merged into one badge: a site
// can meet the target only because unreported time was assumed available.
export function confirmedMeetsTargetLabel(confirmedMeetsTarget: boolean): {
  label: string
  tone: StatusTone
} {
  return confirmedMeetsTarget
    ? { label: 'Confirmed meets target', tone: 'success' }
    : { label: 'Confirmed alone misses target', tone: 'warning' }
}
