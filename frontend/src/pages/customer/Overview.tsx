import { useMemo, useState } from 'react'
import {
  ArrowDownTrayIcon,
  ArrowTrendingUpIcon,
  BoltIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  PlusIcon,
  SignalIcon,
} from '@heroicons/react/24/outline'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  FuelHealthCard,
  PerformanceTrendChart,
  StatCard,
} from '../../components/dashboard'
import { FuelIcon } from '../../components/icons'
import {
  AlertBanner,
  Button,
  DataTable,
  Select,
  StatusBadge,
  Tabs,
  type Column,
} from '../../components/ui'
import { ROUTES, siteDetailPath } from '../../constants'
import { usePortfolioOverview, useSitePerformance } from '../../hooks'
import type { Site, SiteStatus, TrendMetric } from '../../types'
import { getSiteStatusTone } from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const STATUS_FILTER_OPTIONS: Array<SiteStatus | 'All statuses'> = [
  'All statuses',
  'Available',
  'Low diesel',
  'Power outage',
  'Update overdue',
]

export function Overview() {
  const { region, setRegion, period, setPeriod } = useDashboardFilters()
  const [metric, setMetric] = useState<TrendMetric>('availability')
  const [statusFilter, setStatusFilter] = useState<SiteStatus | 'All statuses'>('All statuses')

  const overviewQuery = usePortfolioOverview({ region, period })
  const sitesQuery = useSitePerformance({ region, period })

  const filteredSites = useMemo(() => {
    const sites = sitesQuery.data ?? []
    if (statusFilter === 'All statuses') return sites
    return sites.filter((site) => site.status === statusFilter)
  }, [sitesQuery.data, statusFilter])

  const columns = useMemo<Column<Site>[]>(
    () => [
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
    ],
    []
  )

  const summary = overviewQuery.data?.summary
  const trend = overviewQuery.data?.trend ?? []
  const fuelHealth = overviewQuery.data?.fuelHealth ?? {
    totalLitresRemaining: 0,
    sites: [],
    sitesNeedingReplenishment: 0,
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Portfolio intelligence"
        title="Portfolio overview"
        subtitle="A clear view of your sites. Power, network and support in one place."
        actions={
          <>
            <Button variant="outline" icon={ArrowDownTrayIcon}>
              Export report
            </Button>
            <Button icon={PlusIcon}>Create ticket</Button>
          </>
        }
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        period={period}
        onPeriodChange={setPeriod}
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Power consumption"
          value={
            <>
              {(summary?.energyConsumedKwh ?? 0).toLocaleString()}{' '}
              <span className="text-base font-medium text-text-muted">kWh</span>
            </>
          }
          icon={BoltIcon}
          trend={{ icon: ArrowTrendingUpIcon, label: 'Estimated energy', tone: 'positive' }}
          caption="Selected period · approved load profile"
        />
        <StatCard
          label="Diesel remaining"
          value={
            <>
              {(summary?.dieselRemainingLitres ?? 0).toLocaleString()}{' '}
              <span className="text-base font-medium text-text-muted">litres</span>
            </>
          }
          icon={FuelIcon}
          trend={{
            icon: ExclamationTriangleIcon,
            label: `${summary?.sitesBelowDieselThreshold ?? 0} site${summary?.sitesBelowDieselThreshold === 1 ? '' : 's'} below 25%`,
            tone: 'caution',
          }}
          caption="Latest recorded tank levels"
        />
        <StatCard
          label="Power availability"
          value={
            <>
              {(summary?.powerAvailabilityPercent ?? 0).toFixed(2)}
              <span className="text-base font-medium text-text-muted">%</span>
            </>
          }
          icon={CheckCircleIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: `${summary?.sitesReportedAvailableCount ?? 0} sites last reported available`,
            tone: 'positive',
          }}
          caption="Selected period · reported service"
        />
        <StatCard
          label="Radio network availability"
          value={
            <>
              {(summary?.radioAvailabilityPercent ?? 0).toFixed(2)}
              <span className="text-base font-medium text-text-muted">%</span>
            </>
          }
          icon={SignalIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: `${summary?.sitesWithRadioDataCount ?? 0} of ${summary?.totalSitesInView ?? 0} sites with network data`,
            tone: 'positive',
          }}
          caption="Customer network report · separate source"
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="min-w-0 rounded-lg border border-border bg-surface p-5 lg:col-span-2">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h2 className="text-base font-semibold text-text">Performance over time</h2>
              <p className="text-sm text-text-muted">Today · 24 Oct 2026 · 8 sites in view</p>
            </div>
            <Tabs<TrendMetric>
              aria-label="Chart metric"
              items={[
                { value: 'availability', label: 'Availability' },
                { value: 'energy', label: 'Energy' },
              ]}
              value={metric}
              onChange={setMetric}
            />
          </div>

          <div className="mt-4">
            <PerformanceTrendChart data={trend} metric={metric} />
          </div>

          <div className="mt-2 flex flex-col gap-1 text-xs text-text-muted sm:flex-row sm:items-center sm:justify-between">
            <span>Portfolio trend · equal eligible service windows</span>
            <span>Data details</span>
          </div>
        </div>

        <FuelHealthCard summary={fuelHealth} />
      </div>

      <AlertBanner
        tone="warning"
        icon={ExclamationTriangleIcon}
        title="2 sites need attention."
        description="An active power fault is being addressed."
        action={{ label: 'View issues', to: ROUTES.CUSTOMER_SITES }}
      />

      <div>
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-text">Site performance</h2>
          <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-text-muted">
            {filteredSites.length} {filteredSites.length === 1 ? 'site' : 'sites'}
          </span>
        </div>
        <DataTable
          data={filteredSites}
          columns={columns}
          getRowId={(site) => site.id}
          loading={sitesQuery.isLoading}
          getRowHref={(site) => siteDetailPath(site.id)}
          searchPlaceholder="Search sites or ID..."
          emptyMessage="No sites match this filter."
          headerActions={
            <Select
              value={statusFilter}
              onChange={(event) =>
                setStatusFilter(event.target.value as SiteStatus | 'All statuses')
              }
              aria-label="Filter by status"
              className="w-44"
            >
              {STATUS_FILTER_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </Select>
          }
        />
      </div>
    </div>
  )
}
