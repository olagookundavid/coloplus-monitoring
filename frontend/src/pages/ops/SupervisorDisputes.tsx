import { useMemo, useState } from 'react'
import { ScaleIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import {
  SkeletonDetailPane,
  SkeletonListItems,
  SkeletonStatCards,
} from '../../components/skeletons'
import {
  AttachmentList,
  Button,
  Checkbox,
  ConfirmDialog,
  ConfirmSummary,
  FileDropZone,
  Input,
  Spinner,
  StatusBadge,
  Tabs,
  Textarea,
  type TabItem,
} from '../../components/ui'
import {
  useAttachments,
  useCommentDisputeMutation,
  useDisputeDetail,
  useDisputeSummary,
  useDisputes,
  usePermissions,
  useTransitionDisputeMutation,
} from '../../hooks'
import type { Dispute, DisputeEvent } from '../../types'
import { disputeStatusLabel, formatDateTimeWAT, nameOrId, toast } from '../../utils'

const NO_DISPUTES: Dispute[] = []

// Framed around whose move it is, matching DisputeSummaryResponse's own
// shape (open/withColoplus/withAirtel) rather than inventing a new bucket
// scheme -- a closed dispute has nobody waiting on it.
type Bucket = 'withColoplus' | 'withAirtel' | 'closed'
type TabValue = Bucket | 'all'

function bucketFor(dispute: Dispute): Bucket {
  if (dispute.status === 'closed') return 'closed'
  return dispute.waitingOn === 'airtel' ? 'withAirtel' : 'withColoplus'
}

type PendingAction =
  | { kind: 'startReview'; dispute: Dispute }
  | { kind: 'requestEvidence'; dispute: Dispute; note: string }
  | { kind: 'agreeCorrection'; dispute: Dispute; note: string }
  | { kind: 'resolveNotUpheld'; dispute: Dispute; note: string }
  | { kind: 'backToReview'; dispute: Dispute }
  | { kind: 'resolveUpheld'; dispute: Dispute; note: string; resolutionReportId: string }

interface DetailPaneProps {
  dispute: Dispute
  siteLabel: string
  canManage: boolean
  onRequestAction: (action: PendingAction) => void
}

function FactRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
      <dt className="text-text-muted">{label}</dt>
      <dd className="max-w-[60%] text-right font-medium text-text">{value}</dd>
    </div>
  )
}

function DisputeDetailPane({ dispute, siteLabel, canManage, onRequestAction }: DetailPaneProps) {
  const detailQuery = useDisputeDetail(dispute.id)
  const events = detailQuery.data?.events ?? []
  const attachmentsQuery = useAttachments('dispute', dispute.id)
  const commentMutation = useCommentDisputeMutation()

  const [commentBody, setCommentBody] = useState('')
  const [commentInternal, setCommentInternal] = useState(false)
  const [evidenceNote, setEvidenceNote] = useState('')
  const [correctionNote, setCorrectionNote] = useState('')
  const [notUpheldNote, setNotUpheldNote] = useState('')
  const [upheldNote, setUpheldNote] = useState('')
  const [upheldReportId, setUpheldReportId] = useState('')

  const { label: statusLabelText, tone: statusTone } = disputeStatusLabel(dispute.status)

  function eventBody(event: DisputeEvent): string {
    return event.body || `${event.party === 'airtel' ? 'Airtel' : 'Coloplus'} updated the status`
  }

  async function handleComment() {
    const body = commentBody.trim()
    if (!body) return
    try {
      await commentMutation.mutateAsync({ id: dispute.id, body, internal: commentInternal })
      setCommentBody('')
      setCommentInternal(false)
      toast.success('Comment added.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not add this comment.')
    }
  }

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-text-muted">{dispute.ref}</span>
        <StatusBadge label={statusLabelText} tone={statusTone} />
        {dispute.resolvedAt && !dispute.acknowledgedAt && dispute.status === 'resolved' && (
          <StatusBadge label="Awaiting Airtel's acknowledgement" tone="info" />
        )}
      </div>

      <div>
        <h2 className="text-xl font-bold text-text">
          {dispute.title} · {siteLabel}
        </h2>
        <p className="mt-1 text-sm text-text-muted">{dispute.description}</p>
      </div>

      <dl className="divide-y divide-border rounded-lg border border-border">
        {dispute.raisedBy && (
          <FactRow label="Raised by" value={nameOrId(dispute.raisedByName, dispute.raisedBy)} />
        )}
        <FactRow
          label="Owner"
          value={dispute.ownerId ? nameOrId(dispute.ownerName, dispute.ownerId) : 'Not yet claimed'}
        />
        {dispute.resolvedBy && (
          <FactRow
            label="Resolved by"
            value={nameOrId(dispute.resolvedByName, dispute.resolvedBy)}
          />
        )}
      </dl>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Timeline</p>
        {detailQuery.isLoading ? (
          <Spinner className="mt-3 h-5 w-5" />
        ) : events.length === 0 ? (
          <p className="mt-3 text-sm text-text-muted">No activity yet.</p>
        ) : (
          <ul className="mt-3 flex flex-col gap-3">
            {events.map((event) => (
              <li key={event.id} className="flex gap-3 text-sm">
                <span className="w-32 shrink-0 text-text-muted">
                  {formatDateTimeWAT(event.createdAt)}
                </span>
                <span className="text-text">
                  {eventBody(event)}
                  {event.authorName && (
                    <span className="text-text-muted"> · {event.authorName}</span>
                  )}
                </span>
                {event.internal && <StatusBadge label="Internal" tone="neutral" />}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex flex-col gap-3">
        <AttachmentList
          attachments={attachmentsQuery.data ?? []}
          loading={attachmentsQuery.isLoading}
          emptyMessage="No files attached yet."
        />
        {canManage && dispute.status !== 'closed' && (
          <FileDropZone ownerType="dispute" ownerId={dispute.id} isStaff />
        )}
      </div>

      {!canManage && (
        <p className="text-xs text-text-muted">
          For visibility only — your account doesn't hold dispute-management access.
        </p>
      )}

      {canManage && dispute.status !== 'closed' && (
        <div className="flex flex-col gap-2">
          <div className="flex gap-2">
            <Input
              value={commentBody}
              onChange={(event) => setCommentBody(event.target.value)}
              placeholder="Add a comment"
              aria-label="Add a comment"
              className="flex-1"
            />
            <Button
              variant="secondary"
              onClick={handleComment}
              disabled={!commentBody.trim() || commentMutation.isPending}
              loading={commentMutation.isPending}
            >
              Comment
            </Button>
          </div>
          <Checkbox
            label="Internal — Airtel won't see this"
            checked={commentInternal}
            onChange={(event) => setCommentInternal(event.target.checked)}
          />
        </div>
      )}

      {canManage && dispute.status === 'open' && (
        <Button onClick={() => onRequestAction({ kind: 'startReview', dispute })}>
          Start review
        </Button>
      )}

      {canManage && dispute.status === 'under_review' && (
        <>
          <div className="rounded-lg border border-border bg-surface-subtle p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
              Request evidence
            </p>
            <div className="mt-3 flex flex-col gap-3">
              <Textarea
                value={evidenceNote}
                onChange={(event) => setEvidenceNote(event.target.value)}
                placeholder="What evidence is needed?"
                aria-label="Evidence request note"
              />
              <Button
                variant="secondary"
                disabled={!evidenceNote.trim()}
                onClick={() =>
                  onRequestAction({ kind: 'requestEvidence', dispute, note: evidenceNote.trim() })
                }
                fullWidth
              >
                Request evidence
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-surface-subtle p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
              Agree a correction
            </p>
            <div className="mt-3 flex flex-col gap-3">
              <Textarea
                value={correctionNote}
                onChange={(event) => setCorrectionNote(event.target.value)}
                placeholder="What will be corrected?"
                aria-label="Agreed correction note"
              />
              <Button
                variant="secondary"
                disabled={!correctionNote.trim()}
                onClick={() =>
                  onRequestAction({ kind: 'agreeCorrection', dispute, note: correctionNote.trim() })
                }
                fullWidth
              >
                Agree correction
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-surface-subtle p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
              Resolve — not upheld
            </p>
            <div className="mt-3 flex flex-col gap-3">
              <Textarea
                value={notUpheldNote}
                onChange={(event) => setNotUpheldNote(event.target.value)}
                placeholder="Explain why the figure stands"
                aria-label="Resolution note"
              />
              <Button
                variant="outline"
                disabled={!notUpheldNote.trim()}
                onClick={() =>
                  onRequestAction({
                    kind: 'resolveNotUpheld',
                    dispute,
                    note: notUpheldNote.trim(),
                  })
                }
                fullWidth
              >
                Resolve as not upheld
              </Button>
            </div>
          </div>
        </>
      )}

      {canManage && dispute.status === 'awaiting_evidence' && (
        <Button onClick={() => onRequestAction({ kind: 'backToReview', dispute })}>
          Back to review
        </Button>
      )}

      {canManage && dispute.status === 'agreed_correction' && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Resolve — correction made
          </p>
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={upheldNote}
              onChange={(event) => setUpheldNote(event.target.value)}
              placeholder="Explain the outcome"
              aria-label="Resolution note"
            />
            {dispute.reportId && (
              <Input
                value={upheldReportId}
                onChange={(event) => setUpheldReportId(event.target.value)}
                placeholder="Corrected report id"
                aria-label="Corrected report id"
              />
            )}
            <Button
              disabled={!upheldNote.trim() || (Boolean(dispute.reportId) && !upheldReportId.trim())}
              onClick={() =>
                onRequestAction({
                  kind: 'resolveUpheld',
                  dispute,
                  note: upheldNote.trim(),
                  resolutionReportId: upheldReportId.trim(),
                })
              }
              fullWidth
            >
              Resolve as upheld
            </Button>
          </div>
        </div>
      )}

      {(dispute.status === 'resolved' || dispute.status === 'closed') && dispute.outcome && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Resolution: {dispute.outcome === 'upheld' ? 'Upheld' : 'Not upheld'}
          </p>
          {dispute.resolution && <p className="mt-2 text-sm text-text">{dispute.resolution}</p>}
        </div>
      )}
    </div>
  )
}

export function SupervisorDisputes() {
  const { can } = usePermissions()
  const canManage = can('disputes.manage')

  const disputesQuery = useDisputes()
  const disputes = disputesQuery.data ?? NO_DISPUTES
  const summaryQuery = useDisputeSummary()

  const [tab, setTab] = useState<TabValue>('withColoplus')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null)

  const transitionMutation = useTransitionDisputeMutation()

  const counts = useMemo(() => {
    const result = { withColoplus: 0, withAirtel: 0, closed: 0 }
    for (const dispute of disputes) result[bucketFor(dispute)]++
    return result
  }, [disputes])

  const visibleDisputes = useMemo(
    () => (tab === 'all' ? disputes : disputes.filter((d) => bucketFor(d) === tab)),
    [disputes, tab]
  )

  const selectedDispute =
    visibleDisputes.find((d) => d.id === selectedId) ?? visibleDisputes[0] ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'withColoplus', label: `With you ${counts.withColoplus}` },
    { value: 'withAirtel', label: `With Airtel ${counts.withAirtel}` },
    { value: 'closed', label: `Closed ${counts.closed}` },
    { value: 'all', label: `All ${disputes.length}` },
  ]

  async function handleConfirm() {
    if (!pendingAction) return
    try {
      switch (pendingAction.kind) {
        case 'startReview':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: { status: 'under_review' },
          })
          toast.success('Review started.')
          break
        case 'requestEvidence':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: { status: 'awaiting_evidence', note: pendingAction.note },
          })
          toast.success('Evidence requested.')
          break
        case 'agreeCorrection':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: { status: 'agreed_correction', note: pendingAction.note },
          })
          toast.success('Correction agreed.')
          break
        case 'resolveNotUpheld':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: { status: 'resolved', note: pendingAction.note, outcome: 'not_upheld' },
          })
          toast.success('Dispute resolved.')
          break
        case 'backToReview':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: { status: 'under_review' },
          })
          toast.success('Back under review.')
          break
        case 'resolveUpheld':
          await transitionMutation.mutateAsync({
            id: pendingAction.dispute.id,
            input: {
              status: 'resolved',
              note: pendingAction.note,
              outcome: 'upheld',
              resolutionReportId: pendingAction.resolutionReportId || undefined,
            },
          })
          toast.success('Dispute resolved.')
          break
      }
      setPendingAction(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update this dispute.')
      setPendingAction(null)
    }
  }

  const confirmCopy: Record<PendingAction['kind'], { title: string; description: string }> = {
    startReview: {
      title: 'Start reviewing this dispute?',
      description: 'Claims it as yours and moves it out of the open queue.',
    },
    requestEvidence: {
      title: 'Request evidence from Airtel?',
      description: 'Moves the dispute to "with Airtel" until they reply.',
    },
    agreeCorrection: {
      title: 'Agree this correction?',
      description: 'Commits to the correction described below.',
    },
    resolveNotUpheld: {
      title: 'Resolve as not upheld?',
      description: 'Tells Airtel the figure stands as published.',
    },
    backToReview: {
      title: 'Move this dispute back to review?',
      description: 'Pulls it back from awaiting evidence.',
    },
    resolveUpheld: {
      title: 'Resolve as upheld?',
      description: 'Records the correction as the settlement. This cannot be undone.',
    },
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Review and reconciliation"
        title="Disputes"
        subtitle="Review a disputed figure, request evidence or agree a correction, then resolve it."
      />

      {summaryQuery.isLoading ? (
        <SkeletonStatCards count={3} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <StatCard label="Open" value={summaryQuery.data?.open ?? 0} icon={ScaleIcon} />
          <StatCard
            label="With you"
            value={summaryQuery.data?.withColoplus ?? 0}
            icon={ScaleIcon}
          />
          <StatCard
            label="With Airtel"
            value={summaryQuery.data?.withAirtel ?? 0}
            icon={ScaleIcon}
          />
        </div>
      )}

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {disputesQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : visibleDisputes.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              visibleDisputes.map((dispute) => {
                const selected = dispute.id === (selectedDispute?.id ?? null)
                const { label, tone } = disputeStatusLabel(dispute.status)
                return (
                  <li key={dispute.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(dispute.id)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs text-text-muted">{dispute.ref}</span>
                        <StatusBadge label={label} tone={tone} />
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {dispute.title} · {nameOrId(dispute.siteName, dispute.siteId)}
                      </p>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {disputesQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedDispute ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select a dispute to see its details.
            </div>
          ) : (
            <DisputeDetailPane
              key={selectedDispute.id}
              dispute={selectedDispute}
              siteLabel={nameOrId(selectedDispute.siteName, selectedDispute.siteId)}
              canManage={canManage}
              onRequestAction={setPendingAction}
            />
          )}
        </div>
      </div>

      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={pendingAction ? confirmCopy[pendingAction.kind].title : ''}
        description={pendingAction ? confirmCopy[pendingAction.kind].description : ''}
        tone={
          pendingAction?.kind === 'resolveNotUpheld' || pendingAction?.kind === 'resolveUpheld'
            ? 'success'
            : 'info'
        }
        icon={ScaleIcon}
        confirmLabel="Confirm"
        pending={transitionMutation.isPending}
        body={
          pendingAction && (
            <ConfirmSummary
              title={`${pendingAction.dispute.title} · ${nameOrId(pendingAction.dispute.siteName, pendingAction.dispute.siteId)}`}
              meta={pendingAction.dispute.ref}
            />
          )
        }
        onConfirm={handleConfirm}
        onCancel={() => setPendingAction(null)}
      />
    </div>
  )
}
