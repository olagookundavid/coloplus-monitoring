import type { StatusTone } from '../components/ui'
import type { SiteCondition } from '../types'

export function siteConditionLabel(condition: SiteCondition): { label: string; tone: StatusTone } {
  switch (condition) {
    case 'available':
      return { label: 'Available', tone: 'success' }
    case 'outage_reported':
      return { label: 'Outage reported', tone: 'danger' }
    case 'awaiting_update':
      return { label: 'Awaiting update', tone: 'warning' }
  }
}
