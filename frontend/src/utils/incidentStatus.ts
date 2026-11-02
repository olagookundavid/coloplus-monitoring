import type { StatusTone } from '../components/ui'
import type { IncidentSeverity, IncidentWorkflowStatus } from '../types'

export function severityLabel(severity: IncidentSeverity): { label: string; tone: StatusTone } {
  switch (severity) {
    case 'critical':
      return { label: 'High', tone: 'danger' }
    case 'major':
      return { label: 'Medium', tone: 'warning' }
    case 'minor':
      return { label: 'Low', tone: 'neutral' }
  }
}

export type IncidentBucket = 'open' | 'awaitingReview' | 'closed'

// The real 7-stage lifecycle (domain.IncidentStatus), simplified to the
// three states a reporter needs to act on: restored/verified both mean "my
// part is done, a supervisor still needs to verify and close it."
export function describeIncidentStatus(
  status: IncidentWorkflowStatus
): { bucket: IncidentBucket; label: string; tone: StatusTone } {
  switch (status) {
    case 'restored':
    case 'verified':
      return { bucket: 'awaitingReview', label: 'Awaiting review', tone: 'warning' }
    case 'closed':
      return { bucket: 'closed', label: 'Closed', tone: 'neutral' }
    default:
      return { bucket: 'open', label: 'Open', tone: 'danger' }
  }
}
