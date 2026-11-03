import { useMemo } from 'react'
import {
  BoltIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  FuelHealthCard,
  NO_SITES,
  PerformanceTrendChart,
  periodToDateRange,
  periodToTrendRange,
  SITE_COLUMNS,
  SiteAttentionBanner,
  StatCard,
} from '../../components/dashboard'
import { SkeletonChartRow, SkeletonStatCards } from '../../components/skeletons'
import { Button, DataTable } from '../../components/ui'
import { siteDetailPath } from '../../constants'
import { usePortfolioOverview, usePortfolioTrend } from '../../hooks'
import type { FuelHealthSummary, PortfolioSite } from '../../types'
import { useDashboardFilters } from './CustomerLayout'

function buildFuelHealthSummary(sites: PortfolioSite[]): FuelHealthSummary {
  const withFuel = sites.filter((site) => site.fuel?.litres != null)
  const needingAttention = withFuel.filter((site) => site.fuel?.low || site.fuel?.empty)
  return {
    totalLitresRemaining: Math.round(
      withFuel.reduce((sum, site) => sum + (site.fuel?.litres ?? 0), 0)
    ),
    sites: needingAttention.slice(0, 5).map((site) => ({
      siteId: site.id,
      siteName: site.name,
      litres: site.fuel?.litres ?? null,
      hoursRemaining: site.fuel?.hoursRemaining ?? null,
      low: site.fuel?.low ?? false,
      empty: site.fuel?.empty ?? false,
    })),
    sitesNeedingReplenishment: needingAttention.length,
  }
}

export function Overview() {
  const { region, setRegion, period, setPeriod, openCreateTicket } = useDashboardFilters()

  const overviewQuery = usePortfolioOverview({ region, ...periodToDateRange(period) })
  const trendQuery = usePortfolioTrend({ region, ...periodToTrendRange(period) })
  const sites = overviewQuery.data?.sites ?? NO_SITES

  const fuelHealth = useMemo(() => buildFuelHealthSummary(sites), [sites])

  const { reportedAvailabilityPct, confirmedServedSharePct, assumedAvailableMinutes } =
    overviewQuery.data?.portfolio ?? {}

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Portfolio intelligence"
        title="Portfolio overview"
        subtitle="A clear view of your sites. Power, incidents and support in one place."
        actions={
          <Button icon={PlusIcon} onClick={() => openCreateTicket()}>
            Create ticket
          </Button>
        }
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        regionOptions={overviewQuery.data?.regions ?? []}
        period={period}
        onPeriodChange={setPeriod}
      />

      {overviewQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="Active sites"
            value={overviewQuery.data?.activeSites ?? 0}
            icon={CheckCircleIcon}
            caption={`${overviewQuery.data?.sitesOutageReported ?? 0} with an outage reported`}
          />
          <StatCard
            label="Reported availability"
            value={
              reportedAvailabilityPct == null ? '—' : `${reportedAvailabilityPct.toFixed(2)}%`
            }
            icon={BoltIcon}
            caption={
              assumedAvailableMinutes && assumedAvailableMinutes > 0
                ? 'Includes minutes nobody reported, counted as available by default'
                : 'Confirmed by approved reports'
            }
          />
          <StatCard
            label="Confirmed served share"
            value={
              confirmedServedSharePct == null ? '—' : `${confirmedServedSharePct.toFixed(2)}%`
            }
            icon={CheckCircleIcon}
            caption="Confirmed only — never includes assumed time"
          />
          <StatCard
            label="Data quality"
            value={overviewQuery.data?.dataQuality?.level ?? '—'}
            icon={ExclamationTriangleIcon}
            caption={
              overviewQuery.data?.dataQuality
                ? `${overviewQuery.data.dataQuality.reportedDays}/${overviewQuery.data.dataQuality.days} days reported`
                : undefined
            }
          />
        </div>
      )}

      <SiteAttentionBanner sites={sites} />

      {trendQuery.isLoading ? (
        <SkeletonChartRow />
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div className="min-w-0 rounded-lg border border-border bg-surface p-5 lg:col-span-2">
            <h2 className="text-base font-semibold text-text">Performance over time</h2>
            <p className="mb-4 text-sm text-text-muted">
              Daily availability across your active sites
            </p>
            <PerformanceTrendChart points={trendQuery.data?.points ?? []} />
          </div>
          <FuelHealthCard summary={fuelHealth} />
        </div>
      )}

      <div>
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-text">Site performance</h2>
          <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-text-muted">
            {sites.length} {sites.length === 1 ? 'site' : 'sites'}
          </span>
        </div>
        <DataTable
          data={sites}
          columns={SITE_COLUMNS}
          getRowId={(site) => site.id}
          loading={overviewQuery.isLoading}
          getRowHref={(site) => siteDetailPath(site.id)}
          searchPlaceholder="Search sites or ID..."
          emptyMessage="No sites match this filter."
        />
      </div>
    </div>
  )
}
