import type { DailyReportPanelMode } from '../components/dashboard'
import type { StatusTone } from '../components/ui'
import type { ReportStatus, SiteReportRow } from '../types'

export function statusLabel(status?: ReportStatus): { label: string; tone: StatusTone } {
  if (!status) return { label: 'Not reported', tone: 'danger' }
  if (status === 'draft') return { label: 'Draft saved', tone: 'warning' }
  if (status === 'rejected') return { label: 'Returned', tone: 'danger' }
  return { label: 'Submitted', tone: 'success' } // submitted or approved
}

export function actionFor(row: SiteReportRow): { label: string; mode: DailyReportPanelMode } {
  const status = row.todayStatus?.status
  if (!status) return { label: 'Report', mode: 'create' }
  if (status === 'draft') return { label: 'Continue', mode: 'edit' }
  if (status === 'rejected') return { label: 'Fix', mode: 'edit' }
  return { label: 'View', mode: 'view' }
}
