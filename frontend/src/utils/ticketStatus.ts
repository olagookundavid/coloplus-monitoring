import type { StatusTone } from '../components/ui'
import type { TicketPriority, TicketStatus } from '../types'

export function getTicketStatusTone(status: TicketStatus): StatusTone {
  switch (status) {
    case 'Open':
      return 'warning'
    case 'In progress':
      return 'info'
    case 'Resolved':
      return 'success'
  }
}

export function getTicketPriorityTone(priority: TicketPriority): StatusTone {
  switch (priority) {
    case 'Critical':
      return 'danger'
    case 'High':
      return 'warning'
    case 'Medium':
      return 'info'
    case 'Low':
      return 'neutral'
  }
}
