import { useMemo, useState } from 'react'
import {
  BoltIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  FireIcon,
  MegaphoneIcon,
} from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import {
  SkeletonDetailPane,
  SkeletonListItems,
  SkeletonStatCards,
} from '../../components/skeletons'
import {
  Button,
  ConfirmDialog,
  ConfirmSummary,
  Input,
  Select,
  Spinner,
  StatusBadge,
  Tabs,
  Textarea,
  type StatusTone,
  type TabItem,
} from '../../components/ui'
import { useAuth } from '../../contexts'
import {
  useApproveIncidentCorrectionMutation,
  useAssignIncidentMutation,
  useIncidentCorrections,
  useIncidentSummary,
  useLinkTicketIncidentMutation,
  usePermissions,
  usePublishIncidentMutation,
  useRejectIncidentCorrectionMutation,
  useReporterIncidentDetail,
  useReporterIncidents,
  useTickets,
  useTransitionIncidentMutation,
} from '../../hooks'
import type {
  IncidentCorrection,
  IncidentWorkflowStatus,
  ReporterIncident,
  Ticket,
} from '../../types'
import {
  correctionFieldLabel,
  formatAuditWhen,
  formatDateTimeWAT,
  nameOrId,
  severityLabel,
  ticketStatusLabel,
  toast,
} from '../../utils'

const NO_INCIDENTS: ReporterIncident[] = []
const NO_CORRECTIONS: IncidentCorrection[] = []
const NO_TICKETS: Ticket[] = []

// A supervisor-specific framing of the 7-stage lifecycle, distinct from
// utils/incidentStatus.ts's describeIncidentStatus -- that one is explicitly
// written for what a *reporter* needs to act on (restored/verified both read
// as "awaiting review" there); here restored and verified are two different
// actions a supervisor must take (verify, then close).
type SupervisorBucket = 'needsAction' | 'inProgress' | 'verification' | 'closed'
type TabValue = SupervisorBucket | 'all'

function supervisorBucket(status: IncidentWorkflowStatus): SupervisorBucket {
  switch (status) {
    case 'reported':
      return 'needsAction'
    case 'restored':
    case 'verified':
      return 'verification'
    case 'closed':
      return 'closed'
    default:
      return 'inProgress'
  }
}

const STATUS_LABEL: Record<IncidentWorkflowStatus, { label: string; tone: StatusTone }> = {
  reported: { label: 'Reported', tone: 'danger' },
  acknowledged: { label: 'Acknowledged', tone: 'warning' },
  assigned: { label: 'Assigned', tone: 'warning' },
  in_progress: { label: 'In progress', tone: 'warning' },
  restored: { label: 'Restored — needs verification', tone: 'info' },
  verified: { label: 'Verified — ready to close', tone: 'info' },
  closed: { label: 'Closed', tone: 'neutral' },
}

type PendingAction =
  | { kind: 'acknowledge'; incident: ReporterIncident }
  | { kind: 'assign'; incident: ReporterIncident; ownerId: string }
  | { kind: 'verify'; incident: ReporterIncident; note: string; atIso: string }
  | { kind: 'close'; incident: ReporterIncident; rootCause: string }
  | { kind: 'publish'; incident: ReporterIncident }
  | { kind: 'approveCorrection'; incident: ReporterIncident; correction: IncidentCorrection }
  | {
      kind: 'rejectCorrection'
      incident: ReporterIncident
      correction: IncidentCorrection
      note: string
    }
  | { kind: 'linkTicket'; incident: ReporterIncident; ticketId: string; ticketLabel: string }

// The transition endpoint requires `at` (when the confirming observation was
// made) to verify -- matches ReporterIncidents.tsx's own restore flow, which
// collects the same "today at this time" shape for the same reason.
function combineTodayWithTime(time: string): string {
  const [h, m] = time.split(':').map(Number)
  const d = new Date()
  d.setHours(h ?? 0, m ?? 0, 0, 0)
  return d.toISOString()
}

interface DetailPaneProps {
  incident: ReporterIncident
  siteLabel: string
  canManage: boolean
  canLinkTickets: boolean
  currentUserId: string | undefined
  pendingCorrections: IncidentCorrection[]
  onRequestAction: (action: PendingAction) => void
}

function IncidentDetailPane({
  incident,
  siteLabel,
  canManage,
  canLinkTickets,
  currentUserId,
  pendingCorrections,
  onRequestAction,
}: DetailPaneProps) {
  const detailQuery = useReporterIncidentDetail(incident.id)
  const updates = detailQuery.data?.updates ?? []

  const linkedTicketsQuery = useTickets({ incidentId: incident.id })
  const linkedTickets = linkedTicketsQuery.data ?? NO_TICKETS
  const siteTicketsQuery = useTickets({ siteId: incident.siteId })
  const siteTickets = siteTicketsQuery.data ?? NO_TICKETS

  const [ownerId, setOwnerId] = useState(incident.ownerId ?? '')
  const [verifyNote, setVerifyNote] = useState('')
  const [verifiedAtTime, setVerifiedAtTime] = useState('')
  const [rootCause, setRootCause] = useState('')
  const [rejectNotes, setRejectNotes] = useState<Record<string, string>>({})
  const [ticketToLink, setTicketToLink] = useState('')

  const { label: statusLabelText, tone: statusTone } = STATUS_LABEL[incident.status]
  const { label: sevLabel, tone: sevTone } = severityLabel(incident.severity)

  const canAssign = !['restored', 'verified', 'closed'].includes(incident.status)
  // The backend refuses to publish a closed incident -- publish while
  // verified, before closing, or it's stuck unpublished forever.
  const canPublish = !incident.publishedAt && incident.status === 'verified'

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge label={sevLabel} tone={sevTone} />
        <span className="text-sm text-text-muted">{incident.ref}</span>
        <StatusBadge label={statusLabelText} tone={statusTone} />
        {incident.overdue && <StatusBadge label="Update overdue" tone="danger" />}
      </div>

      <div>
        <h2 className="text-xl font-bold text-text">
          {incident.title} · {siteLabel}
        </h2>
        <p className="mt-1 text-sm text-text-muted">{incident.description}</p>
      </div>

      <p className="text-xs text-text-muted">
        Owner: {incident.ownerId ? nameOrId(incident.ownerName, incident.ownerId) : 'Unassigned'}
      </p>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Timeline</p>
        {detailQuery.isLoading ? (
          <Spinner className="mt-3 h-5 w-5" />
        ) : updates.length === 0 ? (
          <p className="mt-3 text-sm text-text-muted">No updates yet.</p>
        ) : (
          <ul className="mt-3 flex flex-col gap-3">
            {updates.map((update) => (
              <li key={update.id} className="flex gap-3 text-sm">
                <span className="w-14 shrink-0 text-text-muted">
                  {formatAuditWhen(update.observedAt ?? update.createdAt).time}
                </span>
                <span className="text-text">
                  {update.body || 'Update recorded'}
                  {update.authorName && (
                    <span className="text-text-muted"> — {update.authorName}</span>
                  )}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {pendingCorrections.length > 0 && (
        <div className="flex flex-col gap-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Correction requested
          </p>
          {pendingCorrections.map((correction) => {
            const isSelf = currentUserId != null && correction.requestedBy === currentUserId
            return (
              <div
                key={correction.id}
                className="rounded-lg border border-border bg-surface-subtle p-4"
              >
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-medium text-text">
                    {correctionFieldLabel(correction.field)}
                  </p>
                  <span className="text-xs text-text-muted">{correction.ref}</span>
                </div>
                <div className="mt-2 flex items-center gap-2 text-sm">
                  <span className="text-text-muted line-through">
                    {formatDateTimeWAT(correction.originalValue)}
                  </span>
                  <span className="font-medium text-text">
                    {formatDateTimeWAT(correction.correctedValue)}
                  </span>
                </div>
                <p className="mt-2 text-sm text-text">{correction.reason}</p>
                <p className="mt-1 text-xs text-text-muted">
                  Requested by {nameOrId(correction.requestedByName, correction.requestedBy)}
                </p>

                {canManage && !isSelf && (
                  <div className="mt-3 flex flex-col gap-2">
                    <Button
                      variant="secondary"
                      onClick={() =>
                        onRequestAction({ kind: 'approveCorrection', incident, correction })
                      }
                    >
                      Approve
                    </Button>
                    <Textarea
                      value={rejectNotes[correction.id] ?? ''}
                      onChange={(event) =>
                        setRejectNotes((current) => ({
                          ...current,
                          [correction.id]: event.target.value,
                        }))
                      }
                      placeholder="Reason for rejecting (required)"
                      aria-label={`Reason for rejecting ${correction.ref}`}
                    />
                    <Button
                      variant="outline"
                      disabled={!(rejectNotes[correction.id] ?? '').trim()}
                      onClick={() =>
                        onRequestAction({
                          kind: 'rejectCorrection',
                          incident,
                          correction,
                          note: (rejectNotes[correction.id] ?? '').trim(),
                        })
                      }
                    >
                      Reject
                    </Button>
                  </div>
                )}
                {canManage && isSelf && (
                  <p className="mt-3 text-xs text-text-muted">
                    You requested this — another supervisor must decide it.
                  </p>
                )}
              </div>
            )
          })}
        </div>
      )}

      <div className="flex flex-col gap-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          Linked tickets
        </p>
        {linkedTicketsQuery.isLoading ? (
          <Spinner className="h-5 w-5" />
        ) : linkedTickets.length === 0 ? (
          <p className="text-sm text-text-muted">No tickets linked yet.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {linkedTickets.map((ticket) => {
              const { label, tone } = ticketStatusLabel(ticket.status)
              return (
                <li
                  key={ticket.id}
                  className="flex items-center justify-between gap-2 rounded-lg border border-border bg-surface-subtle p-3 text-sm"
                >
                  <div className="min-w-0">
                    <p className="truncate font-medium text-text">{ticket.subject}</p>
                    <p className="text-xs text-text-muted">{ticket.ref}</p>
                  </div>
                  <StatusBadge label={label} tone={tone} />
                </li>
              )
            })}
          </ul>
        )}

        {canLinkTickets && (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <Select
              value={ticketToLink}
              onChange={(event) => setTicketToLink(event.target.value)}
              aria-label="Ticket to link"
              className="flex-1"
            >
              <option value="">Select a ticket at this site</option>
              {siteTickets.map((ticket) => (
                <option key={ticket.id} value={ticket.id}>
                  {ticket.ref} · {ticket.subject}
                </option>
              ))}
            </Select>
            <Button
              variant="secondary"
              disabled={!ticketToLink}
              onClick={() => {
                const ticket = siteTickets.find((candidate) => candidate.id === ticketToLink)
                if (!ticket) return
                onRequestAction({
                  kind: 'linkTicket',
                  incident,
                  ticketId: ticket.id,
                  ticketLabel: `${ticket.ref} · ${ticket.subject}`,
                })
              }}
            >
              Link
            </Button>
          </div>
        )}
      </div>

      {!canManage && (
        <p className="text-xs text-text-muted">
          For visibility only — your account doesn't hold incident-management access.
        </p>
      )}

      {canManage && incident.status === 'reported' && (
        <Button
          variant="secondary"
          onClick={() => onRequestAction({ kind: 'acknowledge', incident })}
        >
          Acknowledge
        </Button>
      )}

      {canManage && canAssign && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            {incident.ownerId ? 'Reassign' : 'Assign'}
          </p>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
            <Input
              value={ownerId}
              onChange={(event) => setOwnerId(event.target.value)}
              placeholder="Owner's user ID"
              aria-label="Owner user ID"
              className="flex-1"
            />
            <Button
              variant="secondary"
              disabled={!ownerId.trim()}
              onClick={() => onRequestAction({ kind: 'assign', incident, ownerId: ownerId.trim() })}
            >
              {incident.ownerId ? 'Reassign' : 'Assign'}
            </Button>
          </div>
          <p className="mt-2 text-xs text-text-muted">
            No people picker yet — paste the field officer's user ID. Current owner:{' '}
            {incident.ownerId ? nameOrId(incident.ownerName, incident.ownerId) : 'unassigned'}.
          </p>
        </div>
      )}

      {canManage && incident.status === 'restored' && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Verify</p>
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={verifyNote}
              onChange={(event) => setVerifyNote(event.target.value)}
              placeholder="What was observed to confirm restoration?"
              aria-label="Verification note"
            />
            <Input
              type="time"
              value={verifiedAtTime}
              onChange={(event) => setVerifiedAtTime(event.target.value)}
              aria-label="Observed at"
            />
            <Button
              disabled={!verifyNote.trim() || !verifiedAtTime}
              onClick={() =>
                onRequestAction({
                  kind: 'verify',
                  incident,
                  note: verifyNote.trim(),
                  atIso: combineTodayWithTime(verifiedAtTime),
                })
              }
              fullWidth
            >
              Mark verified
            </Button>
          </div>
        </div>
      )}

      {canManage && canPublish && (
        <Button
          variant="outline"
          icon={MegaphoneIcon}
          onClick={() => onRequestAction({ kind: 'publish', incident })}
        >
          Publish to the customer
        </Button>
      )}

      {canManage && incident.status === 'verified' && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Close</p>
          {!incident.publishedAt && (
            <p className="mt-1 text-xs text-warning-700">
              Publish to the customer first if this should be visible to them — a closed incident can't be
              published afterward.
            </p>
          )}
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={rootCause}
              onChange={(event) => setRootCause(event.target.value)}
              placeholder="Root cause"
              aria-label="Root cause"
            />
            <Button
              disabled={!rootCause.trim()}
              onClick={() =>
                onRequestAction({ kind: 'close', incident, rootCause: rootCause.trim() })
              }
              fullWidth
            >
              Close incident
            </Button>
          </div>
        </div>
      )}

      {incident.status === 'closed' && incident.rootCause && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
            Root cause
          </p>
          <p className="mt-2 text-sm text-text">{incident.rootCause}</p>
        </div>
      )}
    </div>
  )
}

export function SupervisorIncidents() {
  const { can } = usePermissions()
  const canManage = can('incidents.manage')
  const canLinkTickets = can('tickets.manage')
  const { user } = useAuth()

  const incidentsQuery = useReporterIncidents()
  const incidents = incidentsQuery.data ?? NO_INCIDENTS
  const summaryQuery = useIncidentSummary()
  const correctionsQuery = useIncidentCorrections()

  const pendingCorrectionsByIncident = useMemo(() => {
    const map = new Map<string, IncidentCorrection[]>()
    for (const correction of correctionsQuery.data ?? []) {
      if (correction.status !== 'requested') continue
      const list = map.get(correction.incidentId) ?? []
      list.push(correction)
      map.set(correction.incidentId, list)
    }
    return map
  }, [correctionsQuery.data])

  const [tab, setTab] = useState<TabValue>('needsAction')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null)

  const assignMutation = useAssignIncidentMutation()
  const transitionMutation = useTransitionIncidentMutation()
  const publishMutation = usePublishIncidentMutation()
  const approveCorrectionMutation = useApproveIncidentCorrectionMutation()
  const rejectCorrectionMutation = useRejectIncidentCorrectionMutation()
  const linkTicketMutation = useLinkTicketIncidentMutation()
  const mutationPending =
    assignMutation.isPending ||
    transitionMutation.isPending ||
    publishMutation.isPending ||
    approveCorrectionMutation.isPending ||
    rejectCorrectionMutation.isPending ||
    linkTicketMutation.isPending

  const counts = useMemo(() => {
    const result = { needsAction: 0, inProgress: 0, verification: 0, closed: 0 }
    for (const incident of incidents) result[supervisorBucket(incident.status)]++
    return result
  }, [incidents])

  const visibleIncidents = useMemo(
    () => (tab === 'all' ? incidents : incidents.filter((i) => supervisorBucket(i.status) === tab)),
    [incidents, tab]
  )

  const selectedIncident =
    visibleIncidents.find((incident) => incident.id === selectedId) ?? visibleIncidents[0] ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'needsAction', label: `Needs attention ${counts.needsAction}` },
    { value: 'inProgress', label: `In progress ${counts.inProgress}` },
    { value: 'verification', label: `Verification ${counts.verification}` },
    { value: 'closed', label: `Closed ${counts.closed}` },
    { value: 'all', label: `All ${incidents.length}` },
  ]

  async function handleConfirm() {
    if (!pendingAction) return
    try {
      switch (pendingAction.kind) {
        case 'acknowledge':
          await transitionMutation.mutateAsync({
            id: pendingAction.incident.id,
            input: { status: 'acknowledged' },
          })
          toast.success('Incident acknowledged.')
          break
        case 'assign':
          await assignMutation.mutateAsync({
            id: pendingAction.incident.id,
            input: { ownerId: pendingAction.ownerId },
          })
          toast.success('Incident assigned.')
          break
        case 'verify':
          await transitionMutation.mutateAsync({
            id: pendingAction.incident.id,
            input: { status: 'verified', note: pendingAction.note, at: pendingAction.atIso },
          })
          toast.success('Incident verified.')
          break
        case 'close':
          await transitionMutation.mutateAsync({
            id: pendingAction.incident.id,
            input: { status: 'closed', rootCause: pendingAction.rootCause },
          })
          toast.success('Incident closed.')
          break
        case 'publish':
          await publishMutation.mutateAsync(pendingAction.incident.id)
          toast.success('Published to the customer.')
          break
        case 'approveCorrection':
          await approveCorrectionMutation.mutateAsync(pendingAction.correction.id)
          toast.success('Correction approved.')
          break
        case 'rejectCorrection':
          await rejectCorrectionMutation.mutateAsync({
            id: pendingAction.correction.id,
            note: pendingAction.note,
          })
          toast.success('Correction rejected.')
          break
        case 'linkTicket':
          await linkTicketMutation.mutateAsync({
            ticketId: pendingAction.ticketId,
            incidentId: pendingAction.incident.id,
          })
          toast.success('Ticket linked.')
          break
      }
      setPendingAction(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not complete that action.')
      setPendingAction(null)
    }
  }

  const confirmCopy: Record<PendingAction['kind'], { title: string; description: string }> = {
    acknowledge: {
      title: 'Acknowledge this incident?',
      description: 'Marks it as seen by Coloplus and moves it out of "Needs attention."',
    },
    assign: {
      title: 'Assign this incident?',
      description: 'Hands ownership to the user ID you entered.',
    },
    verify: {
      title: 'Mark this incident verified?',
      description: 'Confirms the restoration is real. Next step is closing it.',
    },
    close: {
      title: 'Close this incident?',
      description: 'Records the root cause and ends the workflow.',
    },
    publish: {
      title: 'Publish to the customer?',
      description: 'Makes this incident visible to the customer. This cannot be undone.',
    },
    approveCorrection: {
      title: 'Approve this correction?',
      description: 'Replaces the recorded time on the incident with the corrected value.',
    },
    rejectCorrection: {
      title: 'Reject this correction?',
      description: 'The original value stays unchanged.',
    },
    linkTicket: {
      title: 'Link this ticket?',
      description: 'Ties the ticket to this incident. Replaces any existing link on that ticket.',
    },
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Incidents"
        subtitle="Acknowledge, assign, verify and close incidents across your sites."
      />

      {summaryQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <StatCard
            label="Open"
            value={summaryQuery.data?.open ?? 0}
            icon={ExclamationTriangleIcon}
          />
          <StatCard label="Active" value={summaryQuery.data?.active ?? 0} icon={BoltIcon} />
          <StatCard
            label="Critical open"
            value={summaryQuery.data?.critical ?? 0}
            icon={FireIcon}
          />
          <StatCard
            label="Overdue updates"
            value={summaryQuery.data?.overdue ?? 0}
            icon={ClockIcon}
          />
        </div>
      )}

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {incidentsQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : visibleIncidents.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              visibleIncidents.map((incident) => {
                const selected = incident.id === (selectedIncident?.id ?? null)
                const { label: sevLabel, tone: sevTone } = severityLabel(incident.severity)
                const { label: statusText, tone: statusTone } = STATUS_LABEL[incident.status]
                const { day } = formatAuditWhen(incident.startedAt)
                return (
                  <li key={incident.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(incident.id)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <StatusBadge label={sevLabel} tone={sevTone} />
                        <span className="shrink-0 text-xs text-text-muted">{incident.ref}</span>
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {incident.title} · {nameOrId(incident.siteName, incident.siteId)}
                      </p>
                      <p className="mt-1 text-sm text-text-muted">
                        {day} · <StatusBadge label={statusText} tone={statusTone} />
                      </p>
                      <p className="mt-1 text-xs text-text-muted">
                        Owner:{' '}
                        {incident.ownerId
                          ? nameOrId(incident.ownerName, incident.ownerId)
                          : 'Unassigned'}
                      </p>
                      {pendingCorrectionsByIncident.has(incident.id) && (
                        <p className="mt-1.5">
                          <StatusBadge label="Correction requested" tone="info" />
                        </p>
                      )}
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {incidentsQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedIncident ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select an incident to see its details.
            </div>
          ) : (
            <IncidentDetailPane
              key={selectedIncident.id}
              incident={selectedIncident}
              siteLabel={nameOrId(selectedIncident.siteName, selectedIncident.siteId)}
              canManage={canManage}
              canLinkTickets={canLinkTickets}
              currentUserId={user?.id}
              pendingCorrections={
                pendingCorrectionsByIncident.get(selectedIncident.id) ?? NO_CORRECTIONS
              }
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
          pendingAction?.kind === 'publish' || pendingAction?.kind === 'rejectCorrection'
            ? 'warning'
            : 'success'
        }
        icon={CheckCircleIcon}
        confirmLabel="Confirm"
        pending={mutationPending}
        body={
          pendingAction && (
            <ConfirmSummary
              title={`${pendingAction.incident.title} · ${nameOrId(pendingAction.incident.siteName, pendingAction.incident.siteId)}`}
              meta={pendingAction.incident.ref}
              rows={
                pendingAction.kind === 'approveCorrection' ||
                pendingAction.kind === 'rejectCorrection'
                  ? [
                      {
                        label: correctionFieldLabel(pendingAction.correction.field),
                        value: formatDateTimeWAT(pendingAction.correction.correctedValue),
                      },
                    ]
                  : pendingAction.kind === 'linkTicket'
                    ? [{ label: 'Ticket', value: pendingAction.ticketLabel }]
                    : undefined
              }
            />
          )
        }
        onConfirm={handleConfirm}
        onCancel={() => setPendingAction(null)}
      />
    </div>
  )
}
