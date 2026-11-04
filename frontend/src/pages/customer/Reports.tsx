import {
  ArrowDownTrayIcon,
  CalendarDaysIcon,
  ChartBarIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  DocumentTextIcon,
  InformationCircleIcon,
} from '@heroicons/react/24/outline'

import { useMemo, useState } from 'react'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  periodToDateRange,
  ReportCard,
} from '../../components/dashboard'
import { AlertBanner, DataTable, Input, RowActions, type Column } from '../../components/ui'
import {
  useAdminSites,
  useDownloadCommissioningBacklogMutation,
  useDownloadDailySummariesMutation,
  useDownloadIncidentRegisterMutation,
  useDownloadIssuedExportUrl,
  useDownloadMonthlyPerformanceMutation,
  useDownloadWeeklyExceptionsMutation,
  useIssuedExports,
} from '../../hooks'
import type { IssuedExport } from '../../types'
import {
  formatDateTimeWAT,
  formatFileSize,
  issuedExportKindLabel,
  mondayOf,
  toast,
} from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const NO_ISSUED_EXPORTS: IssuedExport[] = []

const EXPORT_DETAILS = [
  { label: 'Reporting scope', value: 'Site ids, region and the selected period' },
  { label: 'Data context', value: 'Units, source basis, review state and formula version' },
  { label: 'Missing data', value: 'Blank values remain unavailable, never shown as zero' },
  { label: 'Time standard', value: 'Africa/Lagos · WAT (UTC+1)' },
]

export function Reports() {
  const { region, period, setRegion, setPeriod } = useDashboardFilters()
  const sitesQuery = useAdminSites()
  const regionOptions = useMemo(
    () => [...new Set((sitesQuery.data ?? []).map((site) => site.region))].sort(),
    [sitesQuery.data]
  )
  const dailySummariesMutation = useDownloadDailySummariesMutation()
  const incidentRegisterMutation = useDownloadIncidentRegisterMutation()

  const { date, from, to } = periodToDateRange(period)
  const range = { from: from ?? date ?? '', to: to ?? date ?? '', region: region || undefined }

  async function handleDownload(mutation: typeof dailySummariesMutation, label: string) {
    try {
      await mutation.mutateAsync(range)
      toast.success(`${label} downloaded.`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Could not generate ${label.toLowerCase()}.`)
    }
  }

  const [month, setMonth] = useState('')
  const monthlyPerformanceMutation = useDownloadMonthlyPerformanceMutation()

  async function handleDownloadMonthly() {
    if (!month) return
    try {
      await monthlyPerformanceMutation.mutateAsync({ month, region: region || undefined })
      toast.success('Monthly performance downloaded.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not generate monthly performance.')
    }
  }

  const [weekDate, setWeekDate] = useState('')
  const weeklyExceptionsMutation = useDownloadWeeklyExceptionsMutation()

  async function handleDownloadWeekly() {
    if (!weekDate) return
    try {
      await weeklyExceptionsMutation.mutateAsync({
        weekStart: mondayOf(weekDate),
        region: region || undefined,
      })
      toast.success('Weekly exceptions downloaded.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not generate weekly exceptions.')
    }
  }

  const commissioningBacklogMutation = useDownloadCommissioningBacklogMutation()

  async function handleDownloadBacklog() {
    try {
      await commissioningBacklogMutation.mutateAsync({ region: region || undefined })
      toast.success('Commissioning backlog downloaded.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not generate commissioning backlog.')
    }
  }

  const issuedExportsQuery = useIssuedExports()
  const issuedExports = issuedExportsQuery.data?.exports ?? NO_ISSUED_EXPORTS
  const downloadIssuedExportMutation = useDownloadIssuedExportUrl()

  async function handleDownloadIssued(row: IssuedExport) {
    try {
      await downloadIssuedExportMutation.mutateAsync(row.id)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not download this export.')
    }
  }

  const issuedExportColumns: Column<IssuedExport>[] = [
    {
      header: 'Export',
      accessor: (row) => row.filename,
      cell: (row) => (
        <div>
          <p className="font-medium text-text">{issuedExportKindLabel(row.kind)}</p>
          <p className="text-xs text-text-muted">{row.filename}</p>
        </div>
      ),
    },
    {
      header: 'Issued at',
      accessor: (row) => row.createdAt,
      sortable: true,
      cell: (row) => formatDateTimeWAT(row.createdAt),
    },
    {
      header: 'Size',
      accessor: (row) => row.sizeBytes,
      cell: (row) => formatFileSize(row.sizeBytes),
      align: 'right',
    },
    {
      header: 'Actions',
      accessor: () => null,
      align: 'right',
      cell: (row) => (
        <RowActions
          label={`Actions for ${row.filename}`}
          actions={[
            {
              label: 'Download',
              icon: ArrowDownTrayIcon,
              disabled: !row.stored,
              hint: !row.stored ? "The exact file wasn't kept" : undefined,
              onClick: () => void handleDownloadIssued(row),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Performance & insights"
        title="Reports & exports"
        subtitle="Your operational performance, ready to review and share."
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        regionOptions={regionOptions}
        period={period}
        onPeriodChange={setPeriod}
      />

      <AlertBanner
        tone="info"
        icon={InformationCircleIcon}
        title={
          <>
            Exports use <strong>{region || 'all regions'}</strong> and <strong>{period}</strong>.
          </>
        }
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <ReportCard
          icon={DocumentTextIcon}
          title="Daily summaries"
          description="One row per site/day: minutes, coverage, reported availability, provenance and review state."
          onDownload={() => handleDownload(dailySummariesMutation, 'Daily summaries')}
        />
        <ReportCard
          icon={ClipboardDocumentListIcon}
          title="Incident register"
          description="Published incidents in the period — category, severity, times and restoration status."
          onDownload={() => handleDownload(incidentRegisterMutation, 'Incident register')}
        />
        <ReportCard
          icon={ChartBarIcon}
          title="Monthly performance"
          description="A full calendar month's figures per site, capped to today if the month hasn't finished."
          control={
            <Input
              type="month"
              value={month}
              onChange={(event) => setMonth(event.target.value)}
              aria-label="Month"
            />
          }
          disabled={!month}
          onDownload={() => void handleDownloadMonthly()}
        />
        <ReportCard
          icon={CalendarDaysIcon}
          title="Weekly exceptions"
          description="One week's exceptions, from the Monday that week starts on."
          control={
            <div className="flex flex-col gap-1">
              <Input
                type="date"
                value={weekDate}
                onChange={(event) => setWeekDate(event.target.value)}
                aria-label="A day in the week"
              />
              {weekDate && <p className="text-xs text-text-muted">Week of {mondayOf(weekDate)}</p>}
            </div>
          }
          disabled={!weekDate}
          onDownload={() => void handleDownloadWeekly()}
        />
        <ReportCard
          icon={ClipboardDocumentCheckIcon}
          title="Commissioning backlog"
          description="Sites still awaiting acceptance today, longest waiting first."
          onDownload={() => void handleDownloadBacklog()}
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

      <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
        <h2 className="text-base font-semibold text-text">Your past exports</h2>
        <p className="mt-1 text-sm text-text-muted">
          Every export you've downloaded, with its fingerprint. Showing the most recent 200.
        </p>
        <DataTable
          className="mt-4"
          data={issuedExports}
          columns={issuedExportColumns}
          getRowId={(row) => row.id}
          loading={issuedExportsQuery.isLoading}
          searchable={false}
          compact
          emptyMessage="No export downloaded yet."
        />
      </div>
    </div>
  )
}
