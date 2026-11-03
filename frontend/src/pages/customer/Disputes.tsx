import { useMemo, useState } from 'react'
import { ScaleIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, RaiseDisputeModal, StatCard } from '../../components/dashboard'
import { SkeletonDetailPane, SkeletonListItems, SkeletonStatCards } from '../../components/skeletons'
import {
  AttachmentList,
  Button,
  FileDropZone,
  Input,
  Spinner,
  StatusBadge,
  Tabs,
  Textarea,
  type TabItem,
} from '../../components/ui'
import {
  useAdminSites,
  useAttachments,
  useCommentDisputeMutation,
  useDisputeDetail,
  useDisputeSummary,
  useDisputes,
  usePermissions,
  useTransitionDisputeMutation,
} from '../../hooks'
import type { AdminSite, Dispute, DisputeStatus } from '../../types'
import { disputeStatusLabel, formatDateTimeWAT, toast } from '../../utils'

const NO_DISPUTES: Dispute[] = []
const NO_SITES: AdminSite[] = []

type Bucket = 'open' | 'resolved' | 'closed'
type TabValue = Bucket | 'all'

function bucketFor(status: DisputeStatus): Bucket {
  if (status === 'closed') return 'closed'
  if (status === 'resolved') return 'resolved'
  return 'open'
}

function siteLabelFor(sitesById: Map<string, AdminSite>, siteId: string): string {
  return sitesById.get(siteId)?.airtelCode ?? siteId
}

function DisputeDetailPane({
  dispute,
  siteLabel,
  canReview,
}: {
  dispute: Dispute
  siteLabel: string
  canReview: boolean
}) {
  const detailQuery = useDisputeDetail(dispute.id)
  const events = detailQuery.data?.events ?? []
  const [commentBody, setCommentBody] = useState('')
  const [evidenceNote, setEvidenceNote] = useState('')
  const [reopenReason, setReopenReason] = useState('')

  const commentMutation = useCommentDisputeMutation()
  const transitionMutation = useTransitionDisputeMutation()
  const attachmentsQuery = useAttachments('dispute', dispute.id)

  const { label: statusLabel, tone: statusTone } = disputeStatusLabel(dispute.status)
  const canWithdraw = canReview && (dispute.status === 'open' || dispute.status === 'under_review')
  const canSupplyEvidence = canReview && dispute.status === 'awaiting_evidence'
  const canActOnResolved = canReview && dispute.status === 'resolved'
  const canAttach = canReview && dispute.status !== 'closed'

  async function handleComment() {
    const body = commentBody.trim()
    if (!body) return
    try {
      await commentMutation.mutateAsync({ id: dispute.id, body })
      setCommentBody('')
      toast.success('Comment added.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not add this comment.')
    }
  }

  async function handleTransition(status: DisputeStatus, note?: string) {
    try {
      await transitionMutation.mutateAsync({ id: dispute.id, input: { status, note } })
      toast.success('Dispute updated.')
      setEvidenceNote('')
      setReopenReason('')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update this dispute.')
    }
  }

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-text-muted">{dispute.ref}</span>
        <StatusBadge label={statusLabel} tone={statusTone} />
      </div>

      <div>
        <h2 className="text-xl font-bold text-text">
          {dispute.title} · {siteLabel}
        </h2>
        <p className="mt-1 text-sm text-text-muted">{dispute.description}</p>
      </div>

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
                  {event.body || `${event.party === 'airtel' ? 'Airtel' : 'Coloplus'} updated the status`}
                </span>
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
        {canAttach && (
          <FileDropZone ownerType="dispute" ownerId={dispute.id} isStaff={false} />
        )}
      </div>

      {canReview && dispute.status !== 'closed' && (
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
      )}

      {canSupplyEvidence && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Supply evidence
          </p>
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={evidenceNote}
              onChange={(event) => setEvidenceNote(event.target.value)}
              placeholder="Describe the evidence you're providing"
              aria-label="Evidence note"
            />
            <Button
              disabled={!evidenceNote.trim() || transitionMutation.isPending}
              onClick={() => handleTransition('under_review', evidenceNote.trim())}
              fullWidth
            >
              Submit evidence
            </Button>
          </div>
        </div>
      )}

      {canActOnResolved && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Resolution: {dispute.outcome === 'upheld' ? 'Upheld' : 'Not upheld'}
          </p>
          {dispute.resolution && <p className="mt-2 text-sm text-text">{dispute.resolution}</p>}
          <div className="mt-3 flex flex-col gap-3">
            <Button onClick={() => handleTransition('closed')} fullWidth>
              Acknowledge resolution
            </Button>
            <Textarea
              value={reopenReason}
              onChange={(event) => setReopenReason(event.target.value)}
              placeholder="Reason for reopening (required)"
              aria-label="Reopen reason"
            />
            <Button
              variant="outline"
              disabled={!reopenReason.trim()}
              onClick={() => handleTransition('under_review', reopenReason.trim())}
              fullWidth
            >
              Reopen
            </Button>
          </div>
        </div>
      )}

      {canWithdraw && (
        <Button variant="outline" onClick={() => handleTransition('closed')}>
          Withdraw dispute
        </Button>
      )}
    </div>
  )
}

export function Disputes() {
  const { can } = usePermissions()
  const canReview = can('disputes.raise')

  const disputesQuery = useDisputes()
  const disputes = disputesQuery.data ?? NO_DISPUTES
  const summaryQuery = useDisputeSummary()
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.id, site])), [sites])

  const [tab, setTab] = useState<TabValue>('open')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [raiseOpen, setRaiseOpen] = useState(false)

  const counts = useMemo(() => {
    const result = { open: 0, resolved: 0, closed: 0 }
    for (const dispute of disputes) result[bucketFor(dispute.status)]++
    return result
  }, [disputes])

  const visibleDisputes = useMemo(
    () => (tab === 'all' ? disputes : disputes.filter((d) => bucketFor(d.status) === tab)),
    [disputes, tab]
  )

  const selectedDispute =
    visibleDisputes.find((d) => d.id === selectedId) ?? visibleDisputes[0] ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'open', label: `Open ${counts.open}` },
    { value: 'resolved', label: `Resolved ${counts.resolved}` },
    { value: 'closed', label: `Closed ${counts.closed}` },
    { value: 'all', label: `All ${disputes.length}` },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Review and reconciliation"
        title="Disputes"
        subtitle="Question a published figure. Coloplus reviews it; a correction only lands once agreed."
        actions={
          canReview && (
            <Button icon={ScaleIcon} onClick={() => setRaiseOpen(true)}>
              Raise a dispute
            </Button>
          )
        }
      />

      {summaryQuery.isLoading ? (
        <SkeletonStatCards count={3} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <StatCard label="Open" value={summaryQuery.data?.open ?? 0} icon={ScaleIcon} />
          <StatCard label="With Coloplus" value={summaryQuery.data?.withColoplus ?? 0} icon={ScaleIcon} />
          <StatCard label="With Airtel" value={summaryQuery.data?.withAirtel ?? 0} icon={ScaleIcon} />
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
                        {dispute.title} · {siteLabelFor(sitesById, dispute.siteId)}
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
              siteLabel={siteLabelFor(sitesById, selectedDispute.siteId)}
              canReview={canReview}
            />
          )}
        </div>
      </div>

      <RaiseDisputeModal open={raiseOpen} onClose={() => setRaiseOpen(false)} />
    </div>
  )
}
