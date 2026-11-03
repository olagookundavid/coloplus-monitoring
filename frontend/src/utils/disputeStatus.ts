import type { StatusTone } from '../components/ui'
import type { DisputeStatus } from '../types'

export function disputeStatusLabel(status: DisputeStatus): { label: string; tone: StatusTone } {
  switch (status) {
    case 'open':
      return { label: 'Open', tone: 'danger' }
    case 'under_review':
      return { label: 'Under review', tone: 'warning' }
    case 'awaiting_evidence':
      return { label: 'Awaiting evidence', tone: 'warning' }
    case 'agreed_correction':
      return { label: 'Agreed correction', tone: 'info' }
    case 'resolved':
      return { label: 'Resolved — awaiting acknowledgement', tone: 'info' }
    case 'closed':
      return { label: 'Closed', tone: 'neutral' }
  }
}
