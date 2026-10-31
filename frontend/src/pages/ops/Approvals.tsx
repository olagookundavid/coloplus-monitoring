import { useMemo, useState } from 'react'
import {
  ArrowUturnLeftIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import {
  AlertBanner,
  Button,
  ConfirmDialog,
  ConfirmSummary,
  Spinner,
  StatusBadge,
  Tabs,
  Textarea,
} from '../../components/ui'
import { useAuth } from '../../contexts'
import {
  useAdminReports,
  useAdminSites,
  useApproveReportMutation,
  useManagedUsers,
  usePermissions,
  useRejectReportMutation,
  useReportDetail,
} from '../../hooks'
import type { AdminSite, DailyReport, ManagedUser, ReportDetail, ReportStatus } from '../../types'
import { formatAge, formatDateTimeWAT, toast } from '../../utils'

const NO_REPORTS: DailyReport[] = []
const NO_SITES: AdminSite[] = []
const NO_USERS: ManagedUser[] = []

type TypeFilter = 'all' | 'publication' | 'correction'
type PendingAction = { kind: 'approve' | 'reject'; report: DailyReport }

// "Correction" has no separate backend workflow -- it's a report whose
// supersedes_id is set (backend/internal/services/reports.go review()),
// same convention Overview.tsx's approvals preview already uses.
function isCorrection(report: DailyReport): boolean {
  return Boolean(report.supersedesId)
}

function formatPct(value: number | null | undefined): string {
  return value === null || value === undefined ? '—' : `${value.toFixed(1)}%`
}

export function Approvals() {
  const { user } = useAuth()
  const { can } = usePermissions()
  const canReview = can('reports.review')

  const sitesQuery = useAdminSites()
  const usersQuery = useManagedUsers()
  const sites = sitesQuery.data ?? NO_SITES
  const users = usersQuery.data ?? NO_USERS
  const sitesById = useMemo(() => new Map(sites.map((s) => [s.id, s])), [sites])
  const usersById = useMemo(() => new Map(users.map((u) => [u.id, u])), [users])

  const pendingQuery = useAdminReports('submitted')
  const approvedQuery = useAdminReports('approved')
  const returnedQuery = useAdminReports('rejected')

  const [statusTab, setStatusTab] = useState<ReportStatus>('submitted')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null)
  const [comment, setComment] = useState('')
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null)

  const activeQuery =
    statusTab === 'submitted'
      ? pendingQuery
      : statusTab === 'approved'
        ? approvedQuery
        : returnedQuery
  const reports = activeQuery.data?.reports ?? NO_REPORTS

  const filteredReports = useMemo(() => {
    if (typeFilter === 'all') return reports
    if (typeFilter === 'correction') return reports.filter(isCorrection)
    return reports.filter((r) => !isCorrection(r))
  }, [reports, typeFilter])

  function selectStatusTab(status: ReportStatus) {
    setStatusTab(status)
    setSelectedReportId(null)
    setComment('')
  }

  function selectReport(id: string) {
    setSelectedReportId(id)
    setComment('')
  }

  const detailQuery = useReportDetail(selectedReportId)
  const detail = detailQuery.data
  const supersededQuery = useReportDetail(detail?.report.supersedesId)
  const superseded = detail && isCorrection(detail.report) ? supersededQuery.data : undefined

  const approveMutation = useApproveReportMutation()
  const rejectMutation = useRejectReportMutation()
  const mutationPending = approveMutation.isPending || rejectMutation.isPending

  const isSelfSubmitted = Boolean(detail && user && detail.report.authorId === user.id)
  const isPendingReport = detail?.report.status === 'submitted'
  const actionsEnabled = Boolean(detail) && isPendingReport && canReview && !isSelfSubmitted
  const trimmedComment = comment.trim()
  const canReturn = actionsEnabled && trimmedComment.length > 0

  function requestApprove() {
    if (!detail) return
    setPendingAction({ kind: 'approve', report: detail.report })
  }

  function requestReject() {
    if (!detail || trimmedComment.length === 0) return
    setPendingAction({ kind: 'reject', report: detail.report })
  }

  function handleConfirm() {
    if (!pendingAction) return
    const { kind, report } = pendingAction
    const onSettled = {
      onSuccess: () => {
        toast.success(
          kind === 'approve' ? 'Approved — now visible to Airtel.' : 'Returned to the reporter.'
        )
        setPendingAction(null)
        setSelectedReportId(null)
        setComment('')
      },
      onError: (error: unknown) => {
        toast.error(
          error instanceof Error
            ? error.message
            : kind === 'approve'
              ? 'Could not approve this record.'
              : 'Could not return this record.'
        )
        setPendingAction(null)
      },
    }
    if (kind === 'approve') {
      approveMutation.mutate({ id: report.id, comment: trimmedComment || undefined }, onSettled)
    } else {
      rejectMutation.mutate({ id: report.id, reason: trimmedComment }, onSettled)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Approvals"
        subtitle="Review publication and correction requests before they reach Airtel."
      />

      <Tabs<ReportStatus>
        aria-label="Filter by status"
        items={[
          { value: 'submitted', label: `Pending (${pendingQuery.data?.total ?? 0})` },
          { value: 'approved', label: `Approved (${approvedQuery.data?.total ?? 0})` },
          { value: 'rejected', label: `Returned (${returnedQuery.data?.total ?? 0})` },
        ]}
        value={statusTab}
        onChange={selectStatusTab}
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <Tabs<TypeFilter>
            aria-label="Filter by type"
            items={[
              { value: 'all', label: 'All' },
              { value: 'publication', label: 'Publication' },
              { value: 'correction', label: 'Correction' },
            ]}
            value={typeFilter}
            onChange={setTypeFilter}
          />

          <ul className="flex flex-col gap-2">
            {activeQuery.isLoading ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-center">
                <Spinner className="mx-auto h-5 w-5" />
              </li>
            ) : filteredReports.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              filteredReports.map((report) => {
                const correction = isCorrection(report)
                const site = sitesById.get(report.siteId)
                const author = usersById.get(report.authorId)
                const selected = report.id === selectedReportId
                return (
                  <li key={report.id}>
                    <button
                      type="button"
                      onClick={() => selectReport(report.id)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <StatusBadge
                          label={correction ? 'Correction' : 'Publication'}
                          tone={correction ? 'warning' : 'info'}
                        />
                        <span className="shrink-0 text-xs text-text-muted">
                          {formatAge(report.submittedAt)}
                        </span>
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {site?.name ?? report.siteId} · {report.reportDate}
                      </p>
                      <p className="truncate text-sm text-text-muted">
                        {author?.name ?? report.authorId}
                      </p>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {!selectedReportId ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select a request to review it.
            </div>
          ) : detailQuery.isLoading || !detail ? (
            <div className="flex min-h-48 items-center justify-center rounded-lg border border-border bg-surface p-8">
              <Spinner className="h-6 w-6" />
            </div>
          ) : (
            <DetailCard
              detail={detail}
              superseded={superseded}
              site={sitesById.get(detail.report.siteId)}
              author={usersById.get(detail.report.authorId)}
              reviewer={
                detail.report.reviewerId ? usersById.get(detail.report.reviewerId) : undefined
              }
              isSelfSubmitted={isSelfSubmitted}
              canReview={canReview}
              actionsEnabled={actionsEnabled}
              canReturn={canReturn}
              comment={comment}
              onCommentChange={setComment}
              onApprove={requestApprove}
              onReject={requestReject}
            />
          )}
        </div>
      </div>

      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={pendingAction?.kind === 'approve' ? 'Approve this record?' : 'Return this record?'}
        description={
          pendingAction?.kind === 'approve'
            ? isCorrection(pendingAction.report)
              ? 'This approves the correction and supersedes the original record.'
              : 'This publishes the record to Airtel.'
            : 'This sends the record back to the reporter with your comment.'
        }
        body={
          pendingAction && (
            <ConfirmSummary
              title={`${sitesById.get(pendingAction.report.siteId)?.name ?? pendingAction.report.siteId} · ${pendingAction.report.reportDate}`}
              meta={isCorrection(pendingAction.report) ? 'Correction' : 'Publication'}
            />
          )
        }
        confirmLabel={pendingAction?.kind === 'approve' ? 'Approve' : 'Return'}
        tone={pendingAction?.kind === 'approve' ? 'success' : 'warning'}
        icon={pendingAction?.kind === 'approve' ? CheckCircleIcon : ArrowUturnLeftIcon}
        pending={mutationPending}
        onConfirm={handleConfirm}
        onCancel={() => setPendingAction(null)}
      />
    </div>
  )
}

interface DetailCardProps {
  detail: ReportDetail
  superseded?: ReportDetail
  site?: AdminSite
  author?: ManagedUser
  reviewer?: ManagedUser
  isSelfSubmitted: boolean
  canReview: boolean
  actionsEnabled: boolean
  canReturn: boolean
  comment: string
  onCommentChange: (value: string) => void
  onApprove: () => void
  onReject: () => void
}

function DetailCard({
  detail,
  superseded,
  site,
  author,
  reviewer,
  isSelfSubmitted,
  canReview,
  actionsEnabled,
  canReturn,
  comment,
  onCommentChange,
  onApprove,
  onReject,
}: DetailCardProps) {
  const { report, summary, openDisputes } = detail
  const correction = isCorrection(report)
  const isPending = report.status === 'submitted'

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div>
        <StatusBadge
          label={correction ? 'Correction' : 'Publication'}
          tone={correction ? 'warning' : 'info'}
        />
        <h2 className="mt-2 text-lg font-bold text-text">
          {site?.name ?? report.siteId} · {report.reportDate}
        </h2>
        <p className="text-sm text-text-muted">Revision {report.revision}</p>
      </div>

      {isSelfSubmitted && (
        <AlertBanner
          tone="warning"
          icon={ExclamationTriangleIcon}
          title="You submitted this record, so another approver must review it."
        />
      )}
      {!isSelfSubmitted && !canReview && (
        <p className="text-xs text-text-muted">
          For visibility only — your account doesn't hold report-review access.
        </p>
      )}

      <div>
        <h3 className="text-sm font-semibold text-text">
          {correction ? 'What changed' : "What's being published"}
        </h3>
        <dl className="mt-2 divide-y divide-border rounded-lg border border-border">
          {correction ? (
            <>
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
            </>
          ) : (
            <>
              <FactRow label="Eligible minutes" value={String(summary.eligibleMinutes)} />
              <FactRow label="Served minutes" value={String(summary.servedMinutes)} />
              <FactRow label="Outage minutes" value={String(summary.outageMinutes)} />
              <FactRow label="Coverage" value={formatPct(summary.coveragePct)} />
              <FactRow
                label="Reported availability"
                value={formatPct(summary.reportedAvailabilityPct)}
              />
              {report.notes && <FactRow label="Notes" value={report.notes} />}
            </>
          )}
        </dl>
      </div>

      <div>
        <h3 className="text-sm font-semibold text-text">Record facts</h3>
        <dl className="mt-2 divide-y divide-border rounded-lg border border-border">
          <FactRow label="Open disputes on this revision" value={String(openDisputes)} />
          {correction && (
            <>
              <FactRow
                label="Supersedes"
                value={
                  superseded
                    ? `${superseded.report.reportDate} · revision ${superseded.report.revision}`
                    : (report.supersedesId ?? '—')
                }
              />
              {superseded && (
                <FactRow
                  label="Open disputes on the original"
                  value={String(superseded.openDisputes)}
                />
              )}
            </>
          )}
        </dl>
      </div>

      <div>
        <h3 className="text-sm font-semibold text-text">Approval trail</h3>
        <ul className="mt-2 flex flex-col gap-2 text-sm">
          <li>
            <span className="font-medium text-text">Submitted</span>
            <span className="text-text-muted"> · {author?.name ?? report.authorId}</span>
            {report.submittedAt && (
              <span className="text-text-muted"> · {formatDateTimeWAT(report.submittedAt)}</span>
            )}
          </li>
          {!isPending && (
            <li>
              <span className="font-medium text-text">
                {report.status === 'approved' ? 'Approved' : 'Returned'}
              </span>
              <span className="text-text-muted">
                {' '}
                · {reviewer?.name ?? report.reviewerId ?? '—'}
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

      {isPending && (
        <div className="flex flex-col gap-3 border-t border-border pt-5">
          <div>
            <label htmlFor="approval-comment" className="text-sm font-medium text-text">
              Comment
            </label>
            <p className="text-xs text-text-muted">Required when returning</p>
            <Textarea
              id="approval-comment"
              className="mt-1.5"
              value={comment}
              onChange={(event) => onCommentChange(event.target.value)}
              disabled={!actionsEnabled}
            />
          </div>
          <div className="flex flex-wrap gap-3">
            <Button
              variant="success"
              icon={CheckCircleIcon}
              onClick={onApprove}
              disabled={!actionsEnabled}
            >
              Approve
            </Button>
            <Button
              variant="outline"
              icon={ArrowUturnLeftIcon}
              onClick={onReject}
              disabled={!canReturn}
            >
              Return for correction
            </Button>
          </div>
        </div>
      )}
    </div>
  )
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
