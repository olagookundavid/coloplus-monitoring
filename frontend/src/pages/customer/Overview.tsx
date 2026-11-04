import { useState } from 'react'
import {
  BoltIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  NO_SITES,
  PerformanceTrendChart,
  periodToDateRange,
  periodToTrendRange,
  SITE_COLUMNS,
  SiteAttentionBanner,
  StatCard,
} from '../../components/dashboard'
import { SkeletonChartRow, SkeletonStatCards } from '../../components/skeletons'
import { AlertBanner, Button, DataTable, Select, StatusBadge } from '../../components/ui'
import { siteDetailPath } from '../../constants'
import { usePortfolioOverview, usePortfolioTrend } from '../../hooks'
import type { PortfolioTrendQueryParams } from '../../types'
import { confirmedMeetsTargetLabel, formatMinutes, slaStatusLabel } from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const GRAIN_OPTIONS: Array<{ value: NonNullable<PortfolioTrendQueryParams['grain']>; label: string }> = [
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
]

export function Overview() {
  const { region, setRegion, period, setPeriod, quality, setQuality, openCreateTicket } =
    useDashboardFilters()
  const [grain, setGrain] = useState<NonNullable<PortfolioTrendQueryParams['grain']>>('day')

  const overviewQuery = usePortfolioOverview({
    region,
    quality: (quality || undefined) as 'good' | 'fair' | 'poor' | undefined,
    ...periodToDateRange(period),
  })
  const trendQuery = usePortfolioTrend({ region, grain, ...periodToTrendRange(period) })
  const sites = overviewQuery.data?.sites ?? NO_SITES

  const {
    reportedAvailabilityPct,
    confirmedServedSharePct,
    assumedAvailableMinutes,
    excludedMaintenanceMinutes,
    assumedOutageMinutes,
    sourceMinutes,
  } = overviewQuery.data?.portfolio ?? {}
  const sla = overviewQuery.data?.sla

  const availabilityNotes: string[] = []
  if (assumedAvailableMinutes && assumedAvailableMinutes > 0) {
    availabilityNotes.push('Includes minutes nobody reported, counted as available by default')
  } else {
    availabilityNotes.push('Confirmed by approved reports')
  }
  if (excludedMaintenanceMinutes && excludedMaintenanceMinutes > 0) {
    availabilityNotes.push(`${formatMinutes(excludedMaintenanceMinutes)} planned maintenance excluded`)
  }
  if (assumedOutageMinutes && assumedOutageMinutes > 0) {
    availabilityNotes.push(`${formatMinutes(assumedOutageMinutes)} assumed outage`)
  }

  const dataQuality = overviewQuery.data?.dataQuality
  const dataQualityNotes = dataQuality
    ? [
        `${dataQuality.reportedDays}/${dataQuality.days} days reported`,
        `${dataQuality.completenessPct.toFixed(0)}% complete`,
        `${dataQuality.lateReports} late`,
        `${dataQuality.corrections} correction${dataQuality.corrections === 1 ? '' : 's'}`,
        ...(dataQuality.unconfirmedSharePct > 0
          ? [`${dataQuality.unconfirmedSharePct.toFixed(0)}% unconfirmed`]
          : []),
      ].join(' · ')
    : undefined

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
        quality={quality}
        onQualityChange={setQuality}
        isoTimestamp={overviewQuery.data?.generatedAt}
      />

      {quality && (
        <AlertBanner
          tone="info"
          icon={ExclamationTriangleIcon}
          title={`Showing ${quality}-quality sites only.`}
          description="The counts and totals above still cover every site."
        />
      )}

      {overviewQuery.isLoading ? (
        <SkeletonStatCards count={6} />
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
            caption={availabilityNotes.join(' · ')}
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
            value={dataQuality?.level ?? '—'}
            icon={ExclamationTriangleIcon}
            caption={dataQualityNotes}
          />
          <StatCard
            label="Sites without a report"
            value={overviewQuery.data?.siteDaysWithoutReport ?? 0}
            icon={ExclamationTriangleIcon}
            caption="Site-days in this period with no approved report"
          />
          {sla && (
            <StatCard
              label="SLA"
              value={
                <StatusBadge
                  label={slaStatusLabel(sla.status).label}
                  tone={slaStatusLabel(sla.status).tone}
                />
              }
              icon={CheckCircleIcon}
              caption={
                <StatusBadge
                  label={confirmedMeetsTargetLabel(sla.confirmedMeetsTarget).label}
                  tone={confirmedMeetsTargetLabel(sla.confirmedMeetsTarget).tone}
                />
              }
            />
          )}
        </div>
      )}

      <SiteAttentionBanner sites={sites} />

      {trendQuery.isLoading ? (
        <SkeletonChartRow />
      ) : (
        <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold text-text">Performance over time</h2>
              <p className="text-sm text-text-muted">
                Availability across your active sites, by {grain}
              </p>
            </div>
            <div className="w-32">
              <Select
                value={grain}
                onChange={(event) =>
                  setGrain(event.target.value as NonNullable<PortfolioTrendQueryParams['grain']>)
                }
                aria-label="Trend grain"
              >
                {GRAIN_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </div>
          </div>
          <PerformanceTrendChart points={trendQuery.data?.points ?? []} grain={grain} />
          {sourceMinutes && Object.keys(sourceMinutes).length > 0 && (
            <p className="mt-4 border-t border-border pt-3 text-xs text-text-muted">
              {Object.entries(sourceMinutes)
                .map(([source, minutes]) => `${source}: ${formatMinutes(minutes)}`)
                .join(' · ')}{' '}
              this period — sources can overlap, so this does not add up to served time
            </p>
          )}
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
