import {
  ArrowDownTrayIcon,
  BoltIcon,
  InformationCircleIcon,
  TicketIcon,
} from '@heroicons/react/24/outline'

import { DashboardFilterBar, DashboardPageHeader, ReportCard } from '../../components/dashboard'
import { FuelIcon } from '../../components/icons'
import { AlertBanner, Button } from '../../components/ui'
import { useSitePerformance, useTickets } from '../../hooks'
import {
  buildDieselFuelCsv,
  buildFaultsSupportCsv,
  buildSitePerformanceCsv,
  downloadCsv,
} from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const EXPORT_DETAILS = [
  { label: 'Reporting scope', value: 'Site IDs, region and selected reporting period' },
  { label: 'Data context', value: 'Units, source basis and observation timestamps' },
  { label: 'Missing data', value: 'Blank values remain unavailable' },
  { label: 'Time standard', value: 'Africa/Lagos · WAT (UTC+1)' },
]

export function Reports() {
  const { region, setRegion, period, setPeriod } = useDashboardFilters()
  const sitesQuery = useSitePerformance({ region, period })
  const ticketsQuery = useTickets()
  const scope = { region, period }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Performance & insights"
        title="Reports & exports"
        subtitle="Your operational performance, ready to review and share."
        actions={
          <Button variant="outline" icon={ArrowDownTrayIcon}>
            Export report
          </Button>
        }
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        period={period}
        onPeriodChange={setPeriod}
      />

      <AlertBanner
        tone="info"
        icon={InformationCircleIcon}
        title={
          <>
            Exports use <strong>{region}</strong> and <strong>{period}</strong>.
          </>
        }
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <ReportCard
          icon={BoltIcon}
          title="Site performance"
          description="Energy estimates, power and radio availability, diesel levels and data coverage."
          onDownload={() =>
            downloadCsv(
              'coloplus-site-performance.csv',
              buildSitePerformanceCsv(sitesQuery.data ?? [], scope)
            )
          }
        />
        <ReportCard
          icon={FuelIcon}
          title="Diesel & fuel"
          description="Tank capacity, latest recorded volume, fuel percentage and refill indicators."
          onDownload={() =>
            downloadCsv(
              'coloplus-diesel-fuel.csv',
              buildDieselFuelCsv(sitesQuery.data ?? [], scope)
            )
          }
        />
        <ReportCard
          icon={TicketIcon}
          title="Faults & support"
          description="Customer tickets, service impact, priorities, status and recorded updates."
          onDownload={() =>
            downloadCsv(
              'coloplus-faults-support.csv',
              buildFaultsSupportCsv(ticketsQuery.data ?? [], scope)
            )
          }
        />
      </div>

      <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
        <h2 className="text-base font-semibold text-text">Included with every export</h2>
        <dl className="mt-4 divide-y divide-border">
          {EXPORT_DETAILS.map(({ label, value }) => (
            <div
              key={label}
              className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between"
            >
              <dt className="text-sm text-text-muted">{label}</dt>
              <dd className="text-sm font-medium text-text sm:text-right">{value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  )
}
