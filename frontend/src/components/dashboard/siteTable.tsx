import type { Column } from '../ui'
import type { PortfolioSite } from '../../types'
import { formatDateTimeWAT, siteConditionLabel } from '../../utils'
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
    header: 'Reported availability',
    accessor: (site) => site.summary?.reportedAvailabilityPct ?? -1,
    cell: (site) =>
      site.summary?.reportedAvailabilityPct == null ? (
        <span className="text-text-muted">No approved report</span>
      ) : (
        <span>{site.summary.reportedAvailabilityPct.toFixed(2)}%</span>
      ),
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
      return <StatusBadge label={quality.level} tone={tone} />
    },
    sortable: true,
  },
  {
    header: 'Fuel',
    accessor: (site) => site.fuel?.litres ?? -1,
    cell: (site) =>
      !site.fuel || site.fuel.litres == null ? (
        <span className="text-text-muted">—</span>
      ) : (
        <span className={site.fuel.low || site.fuel.empty ? 'text-warning-700' : undefined}>
          {Math.round(site.fuel.litres).toLocaleString()} L
        </span>
      ),
    align: 'right',
    sortable: true,
  },
  {
    header: 'Updated WAT',
    accessor: (site) => site.asOf ?? '',
    cell: (site) => (site.asOf ? formatDateTimeWAT(site.asOf) : '—'),
    sortable: true,
  },
]
