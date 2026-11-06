import { useState } from 'react'
import {
  ArrowDownTrayIcon,
  CalendarDaysIcon,
  ChartBarIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  DocumentTextIcon,
} from '@heroicons/react/24/outline'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  periodToDateRange,
  ReportCard,
} from '../../components/dashboard'
import { DataTable, Input, RowActions, type Column } from '../../components/ui'
import { useAuth } from '../../contexts'
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
import type { AdminSite, IssuedExport } from '../../types'
import {
  describeIssuedExportParams,
  formatDateTimeWAT,
  formatFileSize,
  issuedByLabel,
  issuedExportKindLabel,
  mondayOf,
  toast,
} from '../../utils'

const NO_EXPORTS: IssuedExport[] = []
const NO_SITES: AdminSite[] = []

export function IssuedExports() {
  const { user } = useAuth()
  const exportsQuery = useIssuedExports()
  const exports = exportsQuery.data?.exports ?? NO_EXPORTS
  const downloadMutation = useDownloadIssuedExportUrl()

  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const regionOptions = [...new Set(sites.map((site) => site.region))].sort()

  const [region, setRegion] = useState('')
  const [period, setPeriod] = useState('Today')
  const { date, from, to } = periodToDateRange(period)
  const range = { from: from ?? date ?? '', to: to ?? date ?? '', region: region || undefined }

  const dailySummariesMutation = useDownloadDailySummariesMutation()
  const incidentRegisterMutation = useDownloadIncidentRegisterMutation()

  async function handleDownloadRange(mutation: typeof dailySummariesMutation, label: string) {
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

  async function handleDownload(row: IssuedExport) {
    try {
      await downloadMutation.mutateAsync(row.id)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not download this export.')
    }
  }

  const columns: Column<IssuedExport>[] = [
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
      header: 'Params',
      accessor: (row) => describeIssuedExportParams(row.params, row.siteName),
      cell: (row) => (
        <span className="text-xs text-text-muted">
          {describeIssuedExportParams(row.params, row.siteName)}
        </span>
      ),
    },
    { header: 'Rows', accessor: (row) => row.rowCount, align: 'right' },
    {
      header: 'Size',
      accessor: (row) => row.sizeBytes,
      cell: (row) => formatFileSize(row.sizeBytes),
      align: 'right',
    },
    {
      header: 'Issued by',
      accessor: (row) => issuedByLabel(row.issuedBy, row.issuedByName, user?.id),
    },
    {
      header: 'Issued at',
      accessor: (row) => row.createdAt,
      sortable: true,
      cell: (row) => formatDateTimeWAT(row.createdAt),
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
              onClick: () => void handleDownload(row),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Exports"
        subtitle="Generate a CSV export, or look up one already issued. An administrator sees every past export; everyone else sees only their own."
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        regionOptions={regionOptions}
        period={period}
        onPeriodChange={setPeriod}
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <ReportCard
          icon={DocumentTextIcon}
          title="Daily summaries"
          description="One row per site/day: minutes, coverage, reported availability, provenance and review state."
          onDownload={() => void handleDownloadRange(dailySummariesMutation, 'Daily summaries')}
        />
        <ReportCard
          icon={ClipboardDocumentListIcon}
          title="Incident register"
          description="Published incidents in the period — category, severity, times and restoration status."
          onDownload={() => void handleDownloadRange(incidentRegisterMutation, 'Incident register')}
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

      <div>
        <h2 className="text-base font-semibold text-text">Issued exports</h2>
        <p className="mt-1 text-xs text-text-muted">
          Showing the most recent 200 issued exports. Times shown in Africa/Lagos (WAT).
        </p>
        <DataTable
          className="mt-4"
          data={exports}
          columns={columns}
          getRowId={(row) => row.id}
          loading={exportsQuery.isLoading}
          searchPlaceholder="Search filename..."
          emptyMessage="No export has been issued yet."
        />
      </div>
    </div>
  )
}
