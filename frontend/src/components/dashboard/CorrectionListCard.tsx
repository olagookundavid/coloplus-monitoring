import { StatusBadge } from '../ui'
import type { CorrectionListItem, MySiteRow } from '../../types'
import { correctionFieldLabel, correctionTabInfo, formatAuditWhen, nameOrId } from '../../utils'

export interface CorrectionListCardProps {
  item: CorrectionListItem
  site?: MySiteRow
  selected: boolean
  onSelect: () => void
}

function cardCopy(
  item: CorrectionListItem,
  siteLabel: string
): { badge: string; ref?: string; title: string; sublineText: string } {
  if (item.kind === 'returnedReport') {
    const { time, day } = formatAuditWhen(item.report.reviewedAt ?? item.report.updatedAt)
    return {
      badge: 'Returned report',
      title: `${siteLabel} · Daily report, ${item.report.reportDate}`,
      sublineText: `Returned ${day}, ${time}`,
    }
  }
  if (item.kind === 'reportCorrection') {
    const { day } = formatAuditWhen(item.report.submittedAt ?? item.report.createdAt)
    return {
      badge: 'Correction request',
      title: `${siteLabel} · Daily report correction, ${item.report.reportDate}`,
      sublineText: `Sent ${day}`,
    }
  }
  const { day } = formatAuditWhen(item.correction.createdAt)
  return {
    badge: 'Correction request',
    ref: item.correction.ref,
    title: `${siteLabel} · ${correctionFieldLabel(item.correction.field)}`,
    sublineText: `Sent ${day}`,
  }
}

export function CorrectionListCard({ item, site, selected, onSelect }: CorrectionListCardProps) {
  const siteName =
    item.kind === 'incidentCorrection' ? item.correction.siteName : item.report.siteName
  const siteLabel = nameOrId(siteName ?? site?.customerSiteCode, item.siteId)
  const { badge, ref, title, sublineText } = cardCopy(item, siteLabel)
  const { label: tabLabel, tone } = correctionTabInfo(item.tab)
  const badgeTone = item.kind === 'returnedReport' ? 'warning' : 'info'

  return (
    <button
      type="button"
      onClick={onSelect}
      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
        selected
          ? 'border-primary bg-primary-soft'
          : 'border-border bg-surface hover:bg-surface-subtle'
      }`}
    >
      <div className="flex items-center justify-between gap-2">
        <StatusBadge label={badge} tone={badgeTone} />
        {ref && <span className="shrink-0 text-xs text-text-muted">{ref}</span>}
      </div>
      <p className="mt-2 truncate font-medium text-text">{title}</p>
      <p className="mt-1 text-sm text-text-muted">
        {sublineText} · <StatusBadge label={tabLabel} tone={tone} />
      </p>
    </button>
  )
}
