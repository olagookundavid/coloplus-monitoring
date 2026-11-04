import type { StatusTone } from '../components/ui'
import type { MaintenanceStatus } from '../types'

export function maintenanceStatusLabel(status: MaintenanceStatus): {
  label: string
  tone: StatusTone
} {
  switch (status) {
    case 'requested':
      return { label: 'Requested', tone: 'info' }
    case 'approved':
      return { label: 'Approved', tone: 'success' }
    case 'rejected':
      return { label: 'Rejected', tone: 'danger' }
    case 'cancelled':
      return { label: 'Cancelled', tone: 'neutral' }
  }
}
