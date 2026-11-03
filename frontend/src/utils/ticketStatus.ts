import type { StatusTone } from '../components/ui'
import type { TicketPriority, TicketStatus } from '../types'

export function ticketStatusLabel(status: TicketStatus): { label: string; tone: StatusTone } {
  switch (status) {
    case 'open':
      return { label: 'Open', tone: 'warning' }
    case 'in_progress':
      return { label: 'In progress', tone: 'info' }
    case 'resolved':
      return { label: 'Resolved', tone: 'success' }
  }
}

export function ticketPriorityLabel(priority: TicketPriority): { label: string; tone: StatusTone } {
  switch (priority) {
    case 'critical':
      return { label: 'Critical', tone: 'danger' }
    case 'high':
      return { label: 'High', tone: 'warning' }
    case 'medium':
      return { label: 'Medium', tone: 'info' }
    case 'low':
      return { label: 'Low', tone: 'neutral' }
  }
}
