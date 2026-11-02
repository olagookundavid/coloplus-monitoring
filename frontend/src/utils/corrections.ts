import type { StatusTone } from '../components/ui'
import type {
  CorrectionListItem,
  CorrectionTab,
  DailyReport,
  IncidentCorrection,
  IncidentCorrectionField,
} from '../types'

export function correctionFieldLabel(field: IncidentCorrectionField): string {
  return field === 'started_at' ? 'Outage started' : 'Restoration time'
}

export function correctionTabInfo(tab: CorrectionTab): { label: string; tone: StatusTone } {
  switch (tab) {
    case 'needsAction':
      return { label: 'Needs your action', tone: 'danger' }
    case 'awaitingReview':
      return { label: 'Awaiting review', tone: 'warning' }
    case 'approved':
      return { label: 'Approved', tone: 'success' }
  }
}

function reportCorrectionTab(report: DailyReport): CorrectionTab | null {
  switch (report.status) {
    case 'rejected':
    case 'draft':
      return 'needsAction'
    case 'submitted':
      return 'awaitingReview'
    case 'approved':
      return 'approved'
    default:
      return null
  }
}

function incidentCorrectionTab(correction: IncidentCorrection): CorrectionTab {
  if (correction.status === 'requested') return 'awaitingReview'
  if (correction.status === 'approved') return 'approved'
  // Rejected: resurfaces as something to act on, with a "Request again"
  // action, rather than silently disappearing -- there is no fourth tab.
  return 'needsAction'
}

function activityTime(item: CorrectionListItem): number {
  if (item.kind === 'incidentCorrection') {
    return new Date(item.correction.reviewedAt ?? item.correction.createdAt).getTime()
  }
  return new Date(
    item.report.reviewedAt ?? item.report.submittedAt ?? item.report.updatedAt
  ).getTime()
}

// Classifies the reporter's own reports and incident corrections into one
// merged, tab-bucketed list. A report with no supersedesId and a rejected
// status is a plain returned report; one with supersedesId set is an
// in-flight report correction (the existing StartCorrection mechanism,
// reused as-is -- see services/adminReports.ts startReportCorrection).
// A draft report with supersedesId set is an orphaned correction revision
// (the save/submit step after StartCorrection failed) -- it still needs
// fixing, so it surfaces under needsAction too, not lost.
export function buildCorrectionList(
  reports: DailyReport[],
  corrections: IncidentCorrection[]
): CorrectionListItem[] {
  const items: CorrectionListItem[] = []

  for (const report of reports) {
    if (report.status === 'rejected' && !report.supersedesId) {
      items.push({ id: report.id, kind: 'returnedReport', tab: 'needsAction', siteId: report.siteId, report })
      continue
    }
    if (report.supersedesId) {
      const tab = reportCorrectionTab(report)
      if (tab) items.push({ id: report.id, kind: 'reportCorrection', tab, siteId: report.siteId, report })
    }
  }

  for (const correction of corrections) {
    items.push({
      id: correction.id,
      kind: 'incidentCorrection',
      tab: incidentCorrectionTab(correction),
      siteId: correction.siteId,
      correction,
    })
  }

  return items.sort((a, b) => activityTime(b) - activityTime(a))
}
