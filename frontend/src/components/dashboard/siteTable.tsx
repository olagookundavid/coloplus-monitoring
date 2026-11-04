import type { Column } from '../ui'
import type { PortfolioSite } from '../../types'
import {
  confirmedMeetsTargetLabel,
  formatDateTimeWAT,
  formatMinutes,
  siteConditionLabel,
  slaStatusLabel,
} from '../../utils'
import { StatusBadge } from '../ui'

// Shared across every page that shows the portfolio site table
// (frontend/CLAUDE.md §4: "When a page shows the same columns twice, share
// them"). Real-backend-shaped (PortfolioSite, from GET /dashboard/overview)
// -- no more invented "radio availability" or "energy consumed" columns;
// neither is a real tracked metric.

export const NO_SITES: PortfolioSite[] = []

export const CONDITION_FILTER_OPTIONS: Array<PortfolioSite['condition'] | 'All conditions'> = [
  'All conditions',
  'available',
  'outage_reported',
  'awaiting_update',
]

export const SITE_COLUMNS: Column<PortfolioSite>[] = [
  {
    header: 'Site / Location',
    accessor: (site) => site.name,
    cell: (site) => (
      <div>
        <p className="font-medium text-text">{site.name}</p>
        <p className="text-xs text-text-muted">
          {site.airtelCode} · {site.region}
        </p>
      </div>
    ),
    sortable: true,
  },
  {
    header: 'Condition',
    accessor: (site) => site.condition,
    cell: (site) => {
      const { label, tone } = siteConditionLabel(site.condition)
      return <StatusBadge label={label} tone={tone} />
    },
    sortable: true,
  },
  {
    header: 'Responsible team',
    accessor: (site) => site.responsibleTeam,
    cell: (site) => <span>{site.responsibleTeam || '—'}</span>,
    sortable: true,
  },
  {
    header: 'Reported availability',
    accessor: (site) => site.summary?.reportedAvailabilityPct ?? -1,
    cell: (site) => {
      const summary = site.summary
      if (summary?.reportedAvailabilityPct == null) {
        return <span className="text-text-muted">No approved report</span>
      }
      const notes: string[] = []
      if (summary.confirmedServedSharePct != null) {
        notes.push(`${summary.confirmedServedSharePct.toFixed(1)}% confirmed`)
      }
      if (summary.assumedAvailableMinutes > 0) {
        notes.push(`${formatMinutes(summary.assumedAvailableMinutes)} assumed`)
      }
      if (summary.excludedMaintenanceMinutes > 0) {
        notes.push(`${formatMinutes(summary.excludedMaintenanceMinutes)} maintenance excluded`)
      }
      if (summary.assumedOutageMinutes > 0) {
        notes.push(`${formatMinutes(summary.assumedOutageMinutes)} assumed outage`)
      }
      if (site.daysWithoutReport > 0) {
        notes.push(`${site.daysWithoutReport} day${site.daysWithoutReport === 1 ? '' : 's'} unreported`)
      }
      return (
        <div>
          <p>{summary.reportedAvailabilityPct.toFixed(2)}%</p>
          {notes.length > 0 && <p className="text-xs text-text-muted">{notes.join(' · ')}</p>}
        </div>
      )
    },
    align: 'right',
    sortable: true,
  },
  {
    header: 'Data quality',
    accessor: (site) => site.dataQuality?.level ?? '',
    cell: (site) => {
      const quality = site.dataQuality
      if (!quality) return <span className="text-text-muted">—</span>
      const tone = quality.level === 'good' ? 'success' : quality.level === 'fair' ? 'warning' : 'danger'
      const notes = [
        `${quality.completenessPct.toFixed(0)}% complete`,
        `${quality.lateReports} late`,
        `${quality.corrections} correction${quality.corrections === 1 ? '' : 's'}`,
      ]
      if (quality.unconfirmedSharePct > 0) {
        notes.push(`${quality.unconfirmedSharePct.toFixed(0)}% unconfirmed`)
      }
      return (
        <div>
          <StatusBadge label={quality.level} tone={tone} />
          <p className="mt-1 text-xs text-text-muted">{notes.join(' · ')}</p>
        </div>
      )
    },
    sortable: true,
  },
  {
    header: 'SLA',
    accessor: (site) => site.sla?.status ?? '',
    cell: (site) => {
      if (!site.sla) return <span className="text-text-muted">—</span>
      const status = slaStatusLabel(site.sla.status)
      const confirmed = confirmedMeetsTargetLabel(site.sla.confirmedMeetsTarget)
      return (
        <div className="flex flex-col items-start gap-1">
          <StatusBadge label={status.label} tone={status.tone} />
          <StatusBadge label={confirmed.label} tone={confirmed.tone} />
        </div>
      )
    },
    sortable: true,
  },
  {
    header: 'Updated WAT',
    accessor: (site) => site.asOf ?? '',
    cell: (site) => (site.asOf ? formatDateTimeWAT(site.asOf) : '—'),
    sortable: true,
  },
]
