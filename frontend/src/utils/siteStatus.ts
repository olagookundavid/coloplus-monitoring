import type { StatusTone } from '../components/ui'
import type { SiteCondition, SiteStatus } from '../types'

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

// The site master record's own status (pending acceptance/active/
// suspended) -- distinct from SiteCondition above, which is a live,
// report-derived reading, not a profile field.
export function siteStatusLabel(status: SiteStatus): { label: string; tone: StatusTone } {
  switch (status) {
    case 'active':
      return { label: 'Active', tone: 'success' }
    case 'pending_acceptance':
      return { label: 'Pending acceptance', tone: 'warning' }
    case 'suspended':
      return { label: 'Suspended', tone: 'neutral' }
  }
}
