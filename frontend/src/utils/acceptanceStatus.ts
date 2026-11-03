import type { StatusTone } from '../components/ui'
import type { AcceptanceStage } from '../types'

export function acceptanceStageLabel(stage: AcceptanceStage): { label: string; tone: StatusTone } {
  switch (stage) {
    case 'requested':
      return { label: 'Requested', tone: 'neutral' }
    case 'built':
      return { label: 'Built', tone: 'neutral' }
    case 'ready_for_inspection':
      return { label: 'Ready for inspection', tone: 'warning' }
    case 'inspected':
      return { label: 'Inspected', tone: 'warning' }
    case 'accepted':
      return { label: 'Accepted', tone: 'success' }
    case 'rejected':
      return { label: 'Rejected', tone: 'danger' }
    case 'on_hold':
      return { label: 'On hold', tone: 'neutral' }
  }
}
