import type { Column } from '../ui'
import type { Site, SiteStatus } from '../../types'
import { getSiteStatusTone } from '../../utils'
import { StatusBadge } from '../ui'

// Shared across every page that shows the site table (frontend/CLAUDE.md §4:
// "When a page shows the same columns twice, share them"). First built for
// Overview, reused as-is by Sites.

// Stable empty array (§4: "a module-level const NO_ROWS: Row[] = [], not an
// inline ?? []") so a loading/undefined query result never hands DataTable a
// fresh array reference every render.
export const NO_SITES: Site[] = []

export const STATUS_FILTER_OPTIONS: Array<SiteStatus | 'All statuses'> = [
  'All statuses',
  'Available',
  'Low diesel',
  'Power outage',
  'Update overdue',
]

export const SITE_COLUMNS: Column<Site>[] = [
  {
    header: 'Site / Location',
    accessor: (site) => site.name,
    cell: (site) => (
      <div>
        <p className="font-medium text-text">{site.name}</p>
        <p className="text-xs text-text-muted">
          {site.code} · {site.region}
        </p>
      </div>
    ),
    sortable: true,
  },
  {
    header: 'Site status',
    accessor: (site) => site.status,
    cell: (site) => <StatusBadge label={site.status} tone={getSiteStatusTone(site.status)} />,
    sortable: true,
  },
  {
    header: 'Power source',
    accessor: (site) => site.powerSource,
    sortable: true,
  },
  {
    header: 'Energy',
    accessor: (site) => site.energyKwh,
    cell: (site) => `${site.energyKwh.toLocaleString()} kWh`,
    align: 'right',
    sortable: true,
  },
  {
    header: 'Diesel level',
    accessor: (site) => site.dieselPercent,
    cell: (site) => (
      <div className="flex items-center justify-end gap-2">
        <div className="h-1.5 w-16 overflow-hidden rounded-full bg-surface-subtle">
          <div
            className={`h-full rounded-full ${site.dieselPercent < 25 ? 'bg-warning' : 'bg-primary'}`}
            style={{ width: `${site.dieselPercent}%` }}
          />
        </div>
        <span>{site.dieselPercent}%</span>
      </div>
    ),
    align: 'right',
    sortable: true,
  },
  {
    header: 'Power avail.',
    accessor: (site) => site.powerAvailabilityPercent,
    cell: (site) => (
      <span
        className={site.powerAvailabilityPercent < 98 ? 'text-warning-700' : 'text-primary-700'}
      >
        {site.powerAvailabilityPercent.toFixed(2)}%
      </span>
    ),
    align: 'right',
    sortable: true,
  },
  {
    header: 'Radio avail.',
    accessor: (site) => site.radioAvailabilityPercent,
    cell: (site) =>
      site.radioAvailabilityPercent == null ? (
        <div>
          <p>—</p>
          <p className="text-xs text-text-muted">Awaiting data</p>
        </div>
      ) : (
        <span>{site.radioAvailabilityPercent.toFixed(2)}%</span>
      ),
    align: 'right',
    sortable: true,
    sortValue: (site) => site.radioAvailabilityPercent ?? -1,
  },
  {
    header: 'Updated WAT',
    accessor: (site) => site.updatedAt,
    cell: (site) => {
      const time = new Date(site.updatedAt).toLocaleTimeString('en-GB', {
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'Africa/Lagos',
      })
      return (
        <div>
          <p>{time}</p>
          {site.status === 'Update overdue' && (
            <p className="text-xs text-warning-700">Update overdue</p>
          )}
        </div>
      )
    },
    sortable: true,
  },
]
