import { ClipboardDocumentListIcon, DocumentTextIcon, InformationCircleIcon } from '@heroicons/react/24/outline'

import { useMemo } from 'react'

import { DashboardFilterBar, DashboardPageHeader, periodToDateRange, ReportCard } from '../../components/dashboard'
import { AlertBanner } from '../../components/ui'
import {
  useAdminSites,
  useDownloadDailySummariesMutation,
  useDownloadIncidentRegisterMutation,
} from '../../hooks'
import { toast } from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

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
