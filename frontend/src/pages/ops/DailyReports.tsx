import { useMemo, useState } from 'react'
import { CheckCircleIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { SkeletonDetailPane, SkeletonListItems } from '../../components/skeletons'
import {
  AlertBanner,
  Button,
  Field,
  Input,
  SegmentedToggle,
  Spinner,
  StatusBadge,
  Textarea,
} from '../../components/ui'
import { useDailyReportForm, useMySites } from '../../hooks'
import type { MySiteRow, PowerStatus } from '../../types'
import { actionFor, statusLabel } from '../../utils'

const NO_SITES: MySiteRow[] = []

function formatReportDateLabel(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1)).toLocaleDateString('en-GB', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  })
}

// The site that most needs attention today: never-reported first, then a
// saved draft, then a returned report, else whatever's first.
function pickDefaultSite(sites: MySiteRow[]): MySiteRow | null {
  return (
    sites.find((s) => !s.todayStatus) ??
    sites.find((s) => s.todayStatus?.status === 'draft') ??
    sites.find((s) => s.todayStatus?.status === 'rejected') ??
    sites[0] ??
    null
  )
}

interface DailyReportDetailPaneProps {
  site: MySiteRow
  reportDate: string
}

// Keyed by site id from the parent, so switching the selected site always
// mounts a fresh instance -- useDailyReportForm's internal state (the form
// fields, the "initialized" flag) must reset per site, not persist across a
// live switch the way it never needs to inside DailyReportPanel (there, the
// panel is always closed before another site can be opened).
function DailyReportDetailPane({ site, reportDate }: DailyReportDetailPaneProps) {
  const mode = actionFor(site).mode
  const {
    detailQuery,
    report,
    powerStatus,
    setPowerStatus,
    generatorHours,
    setGeneratorHours,
    notes,
    setNotes,
    error,
    submitting,
    readOnly,
    handleSaveDraft,
    handleSubmit,
  } = useDailyReportForm({ site, reportDate, reportId: site.todayStatus?.reportId, mode })

  const status = report?.status
  const subtitle = [site.siteType, site.generatorCapacity].filter(Boolean).join(' · ')
  const { label, tone } = statusLabel(site.todayStatus?.status)

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm text-text-muted">{formatReportDateLabel(reportDate)}</p>
          <h2 className="mt-1 truncate text-xl font-bold text-text">
            {site.airtelCode} · {site.name}
          </h2>
          {subtitle && <p className="mt-1 text-sm text-text-muted">{subtitle}</p>}
        </div>
        <StatusBadge label={label} tone={tone} className="shrink-0" />
      </div>

      {detailQuery.isLoading ? (
        <Spinner className="h-6 w-6" />
      ) : (
        <>
          {status === 'rejected' && (
            <AlertBanner
              tone="warning"
              icon={ExclamationTriangleIcon}
              title="Reviewer note:"
              description={report?.reviewComment || 'A reason was not given.'}
            />
          )}
          {status === 'submitted' && (
            <AlertBanner
              tone="success"
              icon={CheckCircleIcon}
              title="Submitted. Waiting for supervisor review."
            />
          )}

          {error && <p className="text-sm text-danger">{error}</p>}

          <Field label="1. Grid power today">
            <SegmentedToggle<PowerStatus>
              aria-label="Grid power today"
              value={powerStatus}
              onChange={setPowerStatus}
              disabled={readOnly}
              options={[
                { value: 'served', label: 'Power on all day' },
                { value: 'outage', label: 'Had an outage' },
              ]}
            />
          </Field>

          <Field
            label="2. Generator runtime (hours)"
            htmlFor="daily-generator-hours"
            hint="Fuel use is estimated from this and shown to Airtel as an estimate."
          >
            <Input
              id="daily-generator-hours"
              type="number"
              min={0}
              max={24}
              step={0.5}
              inputMode="decimal"
              placeholder="0 if the generator did not run"
              value={generatorHours}
              onChange={(event) => setGeneratorHours(event.target.value)}
              disabled={readOnly}
            />
          </Field>

          <Field label="3. Notes" htmlFor="daily-report-notes" hint="(optional)">
            <Textarea
              id="daily-report-notes"
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              disabled={readOnly}
            />
          </Field>

          {status !== 'approved' && (
            <div className="rounded-lg border border-border bg-surface-subtle p-4">
              <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
                Before you submit
              </p>
              <p className="mt-2 inline-flex items-center gap-1.5 text-sm text-text">
                {generatorHours.trim() ? (
                  <CheckCircleIcon
                    className="h-4 w-4 shrink-0 text-primary-600"
                    aria-hidden="true"
                  />
                ) : (
                  <ExclamationTriangleIcon
                    className="h-4 w-4 shrink-0 text-warning-700"
                    aria-hidden="true"
                  />
                )}
                Generator runtime entered (0 if none)
              </p>
            </div>
          )}

          <div className="flex flex-col gap-2 border-t border-border pt-4 sm:flex-row sm:justify-end">
            <Button
              variant="secondary"
              onClick={handleSaveDraft}
              loading={submitting === 'draft'}
              disabled={readOnly || submitting !== null}
            >
              Save draft
            </Button>
            <Button
              onClick={handleSubmit}
              loading={submitting === 'submit'}
              disabled={readOnly || submitting !== null}
            >
              Submit report
            </Button>
          </div>
        </>
      )}
    </div>
  )
}

export function DailyReports() {
  const mySitesQuery = useMySites()
  const sites = mySitesQuery.data?.sites ?? NO_SITES
  const today = mySitesQuery.data?.date ?? new Date().toISOString().slice(0, 10)

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const defaultSite = useMemo(() => pickDefaultSite(sites), [sites])
  const selectedSite = sites.find((s) => s.siteId === selectedId) ?? defaultSite

  const submittedCount = sites.filter(
    (s) => s.todayStatus?.status === 'submitted' || s.todayStatus?.status === 'approved'
  ).length

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Daily reports"
        subtitle="Report what happened at each site today. Values from formulas are shown to Airtel as estimates."
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <h2 className="text-sm font-semibold text-text-muted">
            Sites today · {submittedCount} of {sites.length} submitted
          </h2>
          <ul className="flex flex-col gap-2">
            {mySitesQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : sites.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                No sites are assigned to you yet.
              </li>
            ) : (
              sites.map((site) => {
                const selected = site.siteId === (selectedSite?.siteId ?? null)
                const { label, tone } = statusLabel(site.todayStatus?.status)
                return (
                  <li key={site.siteId}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(site.siteId)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="min-w-0">
                          <p className="truncate font-medium text-text">{site.airtelCode}</p>
                          <p className="truncate text-xs text-text-muted">{site.name}</p>
                        </div>
                        <StatusBadge label={label} tone={tone} className="shrink-0" />
                      </div>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {mySitesQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedSite ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              No sites are assigned to you yet.
            </div>
          ) : (
            <DailyReportDetailPane
              key={selectedSite.siteId}
              site={selectedSite}
              reportDate={today}
            />
          )}
        </div>
      </div>
    </div>
  )
}
