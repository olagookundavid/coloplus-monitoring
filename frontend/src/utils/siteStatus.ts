import type { StatusTone } from '../components/ui'
import type { SiteStatus } from '../types'

export function getSiteStatusTone(status: SiteStatus): StatusTone {
  switch (status) {
    case 'Available':
      return 'success'
    case 'Low diesel':
      return 'warning'
    case 'Power outage':
      return 'danger'
    case 'Update overdue':
      return 'neutral'
  }
}
