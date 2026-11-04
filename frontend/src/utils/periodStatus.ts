import type { StatusTone } from '../components/ui'

export function periodStatusLabel(locked: boolean): { label: string; tone: StatusTone } {
  return locked ? { label: 'Closed', tone: 'neutral' } : { label: 'Reopened', tone: 'warning' }
}
