import {
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'

import {
  AlertBanner,
  Button,
  CalculationDetails,
  Spinner,
  StatusBadge,
  type CalculationDetailRow,
} from '../ui'
import { useReportDetail } from '../../hooks'
import type { CorrectionListItem, DailyReport, IncidentCorrection, MySiteRow } from '../../types'
import { correctionFieldLabel, formatDateTimeWAT } from '../../utils'

export interface CorrectionDetailPaneProps {
  item: CorrectionListItem
  site?: MySiteRow
  onFixReport: (report: DailyReport) => void
  onRequestAgain: (correction: IncidentCorrection) => void
}

function reviewerName(site: MySiteRow | undefined): string {
  return site?.supervisorName || 'a supervisor'
}

function DiffRow({
  label,
  before,
  after,
}: {
  label: string
  before?: number | string
  after: number | string
}) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
      <dt className="text-text-muted">{label}</dt>
      <dd className="flex items-center gap-2">
        <span className="text-text-muted line-through">{before ?? '—'}</span>
        <span className="font-medium text-text">{after}</span>
      </dd>
    </div>
  )
}

function FactRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
      <dt className="text-text-muted">{label}</dt>
      <dd className="max-w-[60%] text-right font-medium text-text">{value}</dd>
    </div>
  )
}

function formatPct(value: number | null | undefined): string {
  return value === null || value === undefined ? '—' : `${value.toFixed(1)}%`
}

function ReturnedReportBody({
  item,
  site,
  onFixReport,
}: {
  item: Extract<CorrectionListItem, { kind: 'returnedReport' }>
  site?: MySiteRow
  onFixReport: (report: DailyReport) => void
}) {
  const { report } = item
  return (
    <>
      <AlertBanner
        tone="warning"
        icon={ExclamationTriangleIcon}
        title={`${reviewerName(site)}:`}
        description={report.reviewComment || 'A reason was not given.'}
      />

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">History</p>
        <ul className="mt-2 flex flex-col gap-2 text-sm">
          <li>
            <span className="font-medium text-text">Submitted by you</span>
            {report.submittedAt && (
              <span className="text-text-muted"> · {formatDateTimeWAT(report.submittedAt)}</span>
            )}
            {report.lateReason && (
              <p className="mt-1 text-text-muted">
                Late submission &mdash; &ldquo;{report.lateReason}&rdquo;
              </p>
            )}
          </li>
          <li>
            <span className="font-medium text-text">Returned by {reviewerName(site)}</span>
            {report.reviewedAt && (
              <span className="text-text-muted"> · {formatDateTimeWAT(report.reviewedAt)}</span>
            )}
          </li>
        </ul>
      </div>

      <Button icon={ArrowPathIcon} onClick={() => onFixReport(report)}>
        Fix now
      </Button>
    </>
  )
}

function ReportCorrectionBody({
  item,
  onFixReport,
}: {
  item: Extract<CorrectionListItem, { kind: 'reportCorrection' }>
  onFixReport: (report: DailyReport) => void
}) {
  const { report } = item
  const detailQuery = useReportDetail(report.id)
  const supersededQuery = useReportDetail(report.supersedesId)
  const needsFix = item.tab === 'needsAction'

  if (detailQuery.isLoading || !detailQuery.data) {
    return <Spinner className="h-5 w-5" />
  }
  const { summary, calculation } = detailQuery.data
  const superseded = supersededQuery.data

  const calculationRows: CalculationDetailRow[] = [
    { label: 'Input report', value: `${calculation.inputReportId} · rev ${calculation.inputRevision}` },
    { label: 'Window', value: `${calculation.windowStart} – ${calculation.windowEnd}` },
    { label: 'Computed', value: formatDateTimeWAT(calculation.computedAt) },
  ]

  return (
    <>
      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          What changed
        </p>
        <dl className="mt-2 divide-y divide-border rounded-lg border border-border">
          <DiffRow
            label="Outage minutes"
            before={superseded?.summary.outageMinutes}
            after={summary.outageMinutes}
          />
          <DiffRow
            label="Served minutes"
            before={superseded?.summary.servedMinutes}
            after={summary.servedMinutes}
          />
          <DiffRow
            label="Coverage"
            before={formatPct(superseded?.summary.coveragePct)}
            after={formatPct(summary.coveragePct)}
          />
          <DiffRow
            label="Reported availability"
            before={formatPct(superseded?.summary.reportedAvailabilityPct)}
            after={formatPct(summary.reportedAvailabilityPct)}
          />
          <FactRow label="Reason for correction" value={report.correctionReason || '—'} />
        </dl>
        <CalculationDetails
          className="mt-3"
          formula={calculation.formula}
          formulaVersion={calculation.version}
          explanation={calculation.explanation}
          rows={calculationRows}
          warnings={calculation.warnings}
        />
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">History</p>
        <ul className="mt-2 flex flex-col gap-2 text-sm">
          <li>
            <span className="font-medium text-text">Requested by you</span>
            {report.createdAt && (
              <span className="text-text-muted"> · {formatDateTimeWAT(report.createdAt)}</span>
            )}
          </li>
          {report.status !== 'submitted' && report.status !== 'draft' && (
            <li>
              <span className="font-medium text-text">
                {report.status === 'approved' ? 'Approved' : 'Returned'}
              </span>
              {report.reviewedAt && (
                <span className="text-text-muted"> · {formatDateTimeWAT(report.reviewedAt)}</span>
              )}
              {report.reviewComment && (
                <p className="mt-1 text-text-muted">&ldquo;{report.reviewComment}&rdquo;</p>
              )}
            </li>
          )}
        </ul>
      </div>

      {needsFix && (
        <Button icon={ArrowPathIcon} onClick={() => onFixReport(report)}>
          {report.status === 'draft' ? 'Resume correction' : 'Fix now'}
        </Button>
      )}
    </>
  )
}

function IncidentCorrectionBody({
  item,
  onRequestAgain,
}: {
  item: Extract<CorrectionListItem, { kind: 'incidentCorrection' }>
  onRequestAgain: (correction: IncidentCorrection) => void
}) {
  const { correction } = item

  return (
    <>
      {correction.status === 'approved' && (
        <AlertBanner
          tone="success"
          icon={CheckCircleIcon}
          title="Approved."
          description="The corrected value replaced the original, which stays on this record."
        />
      )}

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          Value to correct
        </p>
        <dl className="mt-2 divide-y divide-border rounded-lg border border-border">
          <DiffRow
            label={correctionFieldLabel(correction.field)}
            before={formatDateTimeWAT(correction.originalValue)}
            after={formatDateTimeWAT(correction.correctedValue)}
          />
        </dl>
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          Reason for the change
        </p>
        <div className="mt-2 rounded-lg border border-border bg-surface-subtle p-3 text-sm text-text">
          {correction.reason}
        </div>
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">History</p>
        <ul className="mt-2 flex flex-col gap-2 text-sm">
          <li>
            <span className="font-medium text-text">Requested by you</span>
            <span className="text-text-muted"> · {formatDateTimeWAT(correction.createdAt)}</span>
          </li>
          {correction.status !== 'requested' && correction.reviewedAt && (
            <li>
              <span className="font-medium text-text">
                {correction.status === 'approved' ? 'Approved' : 'Declined'}
              </span>
              <span className="text-text-muted"> · {formatDateTimeWAT(correction.reviewedAt)}</span>
              {correction.reviewNote && (
                <p className="mt-1 text-text-muted">&ldquo;{correction.reviewNote}&rdquo;</p>
              )}
            </li>
          )}
        </ul>
      </div>

      {correction.status === 'rejected' && (
        <Button icon={ArrowPathIcon} onClick={() => onRequestAgain(correction)}>
          Request again
        </Button>
      )}
    </>
  )
}

export function CorrectionDetailPane({
  item,
  site,
  onFixReport,
  onRequestAgain,
}: CorrectionDetailPaneProps) {
  const siteLabel = site?.airtelCode ?? item.siteId

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge
          label={item.kind === 'returnedReport' ? 'Returned report' : 'Correction request'}
          tone={item.kind === 'returnedReport' ? 'warning' : 'info'}
        />
        {item.kind === 'incidentCorrection' && (
          <span className="text-sm text-text-muted">{item.correction.ref}</span>
        )}
      </div>

      <h2 className="text-xl font-bold text-text">
        {item.kind === 'incidentCorrection'
          ? `${siteLabel} · ${correctionFieldLabel(item.correction.field)}`
          : `${siteLabel} · Daily report, ${item.report.reportDate}`}
      </h2>

      {item.kind === 'returnedReport' && (
        <ReturnedReportBody item={item} site={site} onFixReport={onFixReport} />
      )}
      {item.kind === 'reportCorrection' && (
        <ReportCorrectionBody item={item} onFixReport={onFixReport} />
      )}
      {item.kind === 'incidentCorrection' && (
        <IncidentCorrectionBody item={item} onRequestAgain={onRequestAgain} />
      )}
    </div>
  )
}
