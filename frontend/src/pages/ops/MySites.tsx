import { useMemo, useState } from 'react'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'

import {
  DailyReportPanel,
  type DailyReportPanelMode,
  DashboardPageHeader,
  DataRecencyIndicator,
  FuelPanel,
  MySiteCard,
  SiteProfilePanel,
} from '../../components/dashboard'
import { SkeletonCardGrid } from '../../components/skeletons'
import { Input, Tabs, type TabItem } from '../../components/ui'
import { useMySites, usePermissions } from '../../hooks'
import type { MySiteRow } from '../../types'
import { actionFor } from '../../utils'

const NO_SITES: MySiteRow[] = []

interface ReportTarget {
  site: MySiteRow
  mode: DailyReportPanelMode
}

function buildTypeFilterItems(rows: MySiteRow[]): TabItem<string>[] {
  const counts = new Map<string, number>()
  for (const row of rows) {
    counts.set(row.siteType, (counts.get(row.siteType) ?? 0) + 1)
  }
  const items: TabItem<string>[] = [{ value: 'all', label: `All ${rows.length}` }]
  for (const [type, count] of counts) {
    items.push({ value: type, label: `${type} ${count}` })
  }
  return items
}

// The mockup's regional label ("Abuja North") has no backing field -- sites
// only carry a free-text `state`. When every assigned site shares one state
// that's shown; a mixed portfolio falls back to a neutral phrase rather than
// inventing a region.
function deriveRegionLabel(rows: MySiteRow[]): string {
  const states = new Set(rows.map((row) => row.state).filter(Boolean))
  if (states.size === 1) return [...states][0] as string
  return 'your region'
}

function matchesType(row: MySiteRow, filter: string): boolean {
  return filter === 'all' || row.siteType === filter
}

function matchesSearch(row: MySiteRow, query: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  return row.customerSiteCode.toLowerCase().includes(q) || row.name.toLowerCase().includes(q)
}

export function MySites() {
  const mySitesQuery = useMySites()
  const rows = mySitesQuery.data?.sites ?? NO_SITES
  const today = mySitesQuery.data?.date ?? new Date().toISOString().slice(0, 10)
  const permissions = usePermissions()
  const canSeeFuel = permissions.can(['fuel.record', 'fuel.read'])

  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [reportTarget, setReportTarget] = useState<ReportTarget | null>(null)
  const [detailsSite, setDetailsSite] = useState<MySiteRow | null>(null)
  const [fuelTarget, setFuelTarget] = useState<MySiteRow | null>(null)

  const regionLabel = useMemo(() => deriveRegionLabel(rows), [rows])
  const typeItems = useMemo(() => buildTypeFilterItems(rows), [rows])
  const filtered = useMemo(
    () => rows.filter((row) => matchesType(row, typeFilter) && matchesSearch(row, search)),
    [rows, typeFilter, search]
  )

  function openReport(site: MySiteRow) {
    setReportTarget({ site, mode: actionFor(site).mode })
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="My sites"
        subtitle={`The ${rows.length} ${rows.length === 1 ? 'site' : 'sites'} assigned to you in ${regionLabel}. You only see sites assigned to you.`}
      />

      <DataRecencyIndicator label="Updated" isoTimestamp={mySitesQuery.data?.generatedAt} />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full sm:max-w-xs">
          <MagnifyingGlassIcon
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted"
            aria-hidden="true"
          />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search site code or name"
            className="pl-9"
            aria-label="Search site code or name"
          />
        </div>
        {typeItems.length > 1 && (
          <Tabs
            aria-label="Filter by site type"
            items={typeItems}
            value={typeFilter}
            onChange={setTypeFilter}
          />
        )}
      </div>

      {mySitesQuery.isLoading ? (
        <SkeletonCardGrid count={6} />
      ) : filtered.length === 0 ? (
        <p className="rounded-lg border border-border bg-surface px-4 py-10 text-center text-sm text-text-muted">
          {rows.length === 0 ? 'No sites are assigned to you yet.' : 'No sites match this search.'}
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((row) => (
            <MySiteCard
              key={row.siteId}
              site={row}
              onReport={() => openReport(row)}
              onDetails={() => setDetailsSite(row)}
              onFuel={canSeeFuel ? () => setFuelTarget(row) : undefined}
            />
          ))}
        </div>
      )}

      {reportTarget && (
        <DailyReportPanel
          open
          onClose={() => setReportTarget(null)}
          site={reportTarget.site}
          reportDate={today}
          reportId={reportTarget.site.todayStatus?.reportId}
          mode={reportTarget.mode}
        />
      )}

      <SiteProfilePanel
        site={detailsSite}
        open={Boolean(detailsSite)}
        onClose={() => setDetailsSite(null)}
        onOpenReport={() => detailsSite && openReport(detailsSite)}
      />

      {fuelTarget && (
        <FuelPanel
          open
          onClose={() => setFuelTarget(null)}
          siteId={fuelTarget.siteId}
          siteName={fuelTarget.name}
        />
      )}
    </div>
  )
}
