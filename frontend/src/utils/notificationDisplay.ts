import type { StatusTone } from '../components/ui'
import type { NotificationStatus } from '../types'

const KIND_LABELS: Record<string, string> = {
  'account.invite': 'Account invite',
  'account.reset': 'Password reset',
  'digest.weekly': 'Weekly digest',
  'report.anomaly': 'Report anomaly',
  'report.overdue': 'Report overdue',
  'report.submitted': 'Report submitted',
  'incident.reported': 'Incident reported',
  'incident.assigned': 'Incident assigned',
  'incident.updated': 'Incident updated',
  'incident.published': 'Incident published',
  'incident.critical': 'Critical incident',
  'incident.overdue': 'Incident overdue',
  'dispute.raised': 'Dispute raised',
  'dispute.event': 'Dispute update',
  'dispute.overdue': 'Dispute overdue',
  'ticket.raised': 'Ticket raised',
  'ticket.event': 'Ticket update',
  'fuel.recorded': 'Fuel recorded',
  'fuel.voided': 'Fuel entry voided',
  'fuel.low': 'Low fuel',
  'maintenance.requested': 'Maintenance requested',
  'maintenance.decided': 'Maintenance decided',
  'acceptance.requested': 'Acceptance requested',
  'acceptance.event': 'Acceptance update',
  'acceptance.overdue': 'Acceptance overdue',
  'acceptance.escalated': 'Acceptance escalated',
}

// A new backend kind never breaks this page -- fall back to a generic
// humanization of the dot-separated string instead of a raw, unexplained key.
export function humanizeNotificationKind(kind: string): string {
  const known = KIND_LABELS[kind]
  if (known) return known
  return kind
    .split(/[._]/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

export function notificationStatusLabel(status: NotificationStatus): {
  label: string
  tone: StatusTone
} {
  switch (status) {
    case 'pending':
      return { label: 'Pending', tone: 'neutral' }
    case 'sent':
      return { label: 'Sent', tone: 'success' }
    case 'failed':
      return { label: 'Failed', tone: 'danger' }
  }
}
