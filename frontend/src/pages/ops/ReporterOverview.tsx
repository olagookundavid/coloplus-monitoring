import { useCallback, useMemo, useState } from 'react'
import {
  ArrowUturnLeftIcon,
  BoltIcon,
  CheckCircleIcon,
  ClockIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'

import {
  DailyReportPanel,
  type DailyReportPanelMode,
  DashboardPageHeader,
  DataRecencyIndicator,
  StatCard,
} from '../../components/dashboard'
import { SkeletonStatCards } from '../../components/skeletons'
import {
  AlertBanner,
  Button,
  DataTable,
  StatusBadge,
  Tabs,
  type Column,
  type TabItem,
} from '../../components/ui'
import { useDashboardOverview } from '../../hooks'
import type { SiteReportRow } from '../../types'
import { actionFor, statusLabel } from '../../utils'

const NO_SITES: SiteReportRow[] = []

type FilterChip = 'all' | 'pending' | 'draft' | 'returned' | 'submitted'

function matchesFilter(row: SiteReportRow, filter: FilterChip): boolean {
  const status = row.todayStatus?.status
  switch (filter) {
    case 'all':
      return true
    case 'pending':
      return !status
    case 'draft':
      return status === 'draft'
    case 'returned':
      return status === 'rejected'
    case 'submitted':
      return status === 'submitted' || status === 'approved'
  }
}

// 18:00 Africa/Lagos -- a display-only assumption, not a server-enforced
// cutoff (the PRD leaves per-site cadence/cutoff an open decision).
function cutoffHasPassed(): boolean {
  const wat = Number(
    new Intl.DateTimeFormat('en-GB', {
      hour: 'numeric',
      hourCycle: 'h23',
      timeZone: 'Africa/Lagos',
    }).format(new Date())
  )
  return wat >= 18
}

interface PanelTarget {
  site: SiteReportRow
  mode: DailyReportPanelMode
}

export function ReporterOverview() {
  const today = new Date().toISOString().slice(0, 10)
  const overviewQuery = useDashboardOverview(today)
  const sites = overviewQuery.data?.sites ?? NO_SITES

  const [filter, setFilter] = useState<FilterChip>('all')
  const [panelTarget, setPanelTarget] = useState<PanelTarget | null>(null)

  const openPanel = useCallback((site: SiteReportRow, mode: DailyReportPanelMode) => {
    setPanelTarget({ site, mode })
  }, [])

  const total = sites.length
  const reportedToday = sites.filter(
    (s) => s.todayStatus?.status === 'submitted' || s.todayStatus?.status === 'approved'
  ).length
  const draftCount = sites.filter((s) => s.todayStatus?.status === 'draft').length
  const pendingCount = sites.filter((s) => !s.todayStatus).length
  const returnedSites = sites.filter((s) => s.todayStatus?.status === 'rejected')

  const nextSite =
    sites.find((s) => !s.todayStatus) ??
    sites.find((s) => s.todayStatus?.status === 'draft') ??
    null

  const filterItems: TabItem<FilterChip>[] = [
    { value: 'all', label: `All` },
    { value: 'pending', label: `Pending (${pendingCount})` },
    { value: 'draft', label: `Draft (${draftCount})` },
    { value: 'returned', label: `Returned (${returnedSites.length})` },
    { value: 'submitted', label: `Submitted (${reportedToday})` },
  ]

  const filteredSites = useMemo(
    () => sites.filter((row) => matchesFilter(row, filter)),
    [sites, filter]
  )

  const columns = useMemo<Column<SiteReportRow>[]>(
    () => [
      { header: 'Site', accessor: (row) => row.customerSiteCode },
      {
        header: 'Name',
        accessor: (row) => row.name,
        cell: (row) => (
          <div className="min-w-0">
            <p className="truncate font-medium text-text">{row.name}</p>
            <p className="truncate text-xs text-text-muted">{row.siteType}</p>
          </div>
        ),
      },
      {
        header: 'Yesterday',
        accessor: (row) => statusLabel(row.yesterdayStatus?.status).label,
        cell: (row) => (
          <span className="text-text-muted">{statusLabel(row.yesterdayStatus?.status).label}</span>
        ),
      },
      {
        header: 'Today',
        accessor: (row) => statusLabel(row.todayStatus?.status).label,
        cell: (row) => {
          const { label, tone } = statusLabel(row.todayStatus?.status)
          return <StatusBadge label={label} tone={tone} />
        },
      },
      {
        header: 'Fuel',
        accessor: (row) => row.fuel?.litres ?? -1,
        cell: (row) =>
          !row.fuel || row.fuel.litres == null ? (
            <span className="text-text-muted">—</span>
          ) : (
            <span className="inline-flex items-center gap-1.5">
              <span className={row.fuel.low || row.fuel.empty ? 'text-warning-700' : undefined}>
                {Math.round(row.fuel.litres).toLocaleString()} L
              </span>
              {(row.fuel.low || row.fuel.empty) && (
                <StatusBadge label={row.fuel.empty ? 'Empty' : 'Low'} tone="warning" />
              )}
            </span>
          ),
        align: 'right',
      },
      {
        header: 'Action',
        accessor: () => '',
        align: 'right',
        cell: (row) => {
          const { label, mode } = actionFor(row)
          return (
            <Button
              size="sm"
              variant={mode === 'view' ? 'outline' : 'primary'}
              onClick={() => openPanel(row, mode)}
            >
              {label}
            </Button>
          )
        },
      },
    ],
    [openPanel]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Overview"
        subtitle={`Daily reports for your ${total} assigned ${total === 1 ? 'site' : 'sites'} are due by 18:00 today.`}
        actions={
          <Button
            icon={PlusIcon}
            onClick={() =>
              nextSite && openPanel(nextSite, nextSite.todayStatus ? 'edit' : 'create')
            }
            disabled={!nextSite}
          >
            Report next site
          </Button>
        }
      />

      <DataRecencyIndicator label="Updated" isoTimestamp={overviewQuery.data?.generatedAt} />

      {overviewQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Today's progress"
            value={`${reportedToday} of ${total}`}
            icon={CheckCircleIcon}
            caption={cutoffHasPassed() ? "Today's 18:00 cut-off has passed" : 'Due by 18:00 today'}
          />
          <StatCard
            label="Still to report"
            value={pendingCount + draftCount}
            icon={ClockIcon}
            caption={`${draftCount} saved as draft`}
          />
          <StatCard
            label="Returned to fix"
            value={returnedSites.length}
            icon={ArrowUturnLeftIcon}
            caption={returnedSites.length > 0 ? 'Needs your attention' : 'All clear'}
          />
          <StatCard
            label="Outages this week"
            value={overviewQuery.data?.outagesThisWeek ?? '—'}
            icon={BoltIcon}
            caption="Trailing 7 days"
          />
        </div>
      )}

      {returnedSites.map((site) => (
        <AlertBanner
          key={site.siteId}
          tone="warning"
          title="Returned for correction"
          description={`${site.customerSiteCode} · ${site.todayStatus?.reviewComment || 'A reason was not given.'}`}
          action={{ label: 'Fix now', onClick: () => openPanel(site, 'edit') }}
        />
      ))}

      <div className="flex flex-col gap-3">
        <h2 className="text-base font-semibold text-text">My sites today</h2>
        <Tabs aria-label="Filter sites" items={filterItems} value={filter} onChange={setFilter} />
        <DataTable
          data={filteredSites}
          columns={columns}
          getRowId={(row) => row.siteId}
          loading={overviewQuery.isLoading}
          emptyMessage="No sites match this filter."
          searchable={false}
          paginated={false}
        />
      </div>

      {panelTarget && (
        <DailyReportPanel
          open={Boolean(panelTarget)}
          onClose={() => setPanelTarget(null)}
          site={panelTarget.site}
          reportDate={today}
          reportId={panelTarget.site.todayStatus?.reportId}
          mode={panelTarget.mode}
        />
      )}
    </div>
  )
}
