import { useMemo, useState } from 'react'
import { TicketIcon } from '@heroicons/react/24/outline'

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
  Select,
  Spinner,
  StatusBadge,
  Tabs,
  Textarea,
  type TabItem,
} from '../../components/ui'
import {
  useAttachments,
  useCommentTicketMutation,
  useLinkTicketIncidentMutation,
  usePermissions,
  useReporterIncidents,
  useTicketDetail,
  useTicketSummary,
  useTickets,
  useTransitionTicketMutation,
} from '../../hooks'
import type { ReporterIncident, Ticket, TicketEvent } from '../../types'
import {
  ticketPriorityLabel,
  ticketStatusLabel,
  formatDateTimeWAT,
  nameOrId,
  toast,
} from '../../utils'

const NO_TICKETS: Ticket[] = []
const NO_INCIDENTS: ReporterIncident[] = []

type Bucket = 'open' | 'inProgress' | 'resolved'
type TabValue = Bucket | 'all'

function bucketFor(ticket: Ticket): Bucket {
  if (ticket.status === 'in_progress') return 'inProgress'
  if (ticket.status === 'resolved') return 'resolved'
  return 'open'
}

type PendingAction =
  | { kind: 'start'; ticket: Ticket }
  | { kind: 'resolve'; ticket: Ticket; note: string }
  | { kind: 'reopenInternal'; ticket: Ticket; note: string }
  | { kind: 'linkIncident'; ticket: Ticket; incidentId: string; incidentLabel: string }

interface DetailPaneProps {
  ticket: Ticket
  siteLabel: string
  siteIncidents: ReporterIncident[]
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

function TicketDetailBody({
  ticket,
  siteLabel,
  siteIncidents,
  canManage,
  onRequestAction,
}: DetailPaneProps) {
  const detailQuery = useTicketDetail(ticket.id)
  const events = detailQuery.data?.events ?? []
  const attachmentsQuery = useAttachments('ticket', ticket.id)
  const commentMutation = useCommentTicketMutation()

  const [commentBody, setCommentBody] = useState('')
  const [commentInternal, setCommentInternal] = useState(false)
  const [resolveNote, setResolveNote] = useState('')
  const [reopenNote, setReopenNote] = useState('')
  const [incidentToLink, setIncidentToLink] = useState('')

  const { label: statusLabelText, tone: statusTone } = ticketStatusLabel(ticket.status)
  const { label: priorityLabelText, tone: priorityTone } = ticketPriorityLabel(ticket.priority)

  function eventBody(event: TicketEvent): string {
    return event.body || `${event.party === 'airtel' ? 'Airtel' : 'Coloplus'} updated the status`
  }

  async function handleComment() {
    const body = commentBody.trim()
    if (!body) return
    try {
      await commentMutation.mutateAsync({ id: ticket.id, body, internal: commentInternal })
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
        <span className="text-sm text-text-muted">{ticket.ref}</span>
        <StatusBadge label={statusLabelText} tone={statusTone} />
        <StatusBadge label={priorityLabelText} tone={priorityTone} />
      </div>

      <div>
        <h2 className="text-xl font-bold text-text">
          {ticket.subject} · {siteLabel}
        </h2>
        <p className="mt-1 text-sm text-text-muted">{ticket.description}</p>
      </div>

      <dl className="divide-y divide-border rounded-lg border border-border">
        {ticket.raisedBy && (
          <FactRow label="Raised by" value={nameOrId(ticket.raisedByName, ticket.raisedBy)} />
        )}
        <FactRow
          label="Assigned to"
          value={
            ticket.assigneeId ? nameOrId(ticket.assigneeName, ticket.assigneeId) : 'Unassigned'
          }
        />
        {ticket.resolvedBy && (
          <FactRow label="Resolved by" value={nameOrId(ticket.resolvedByName, ticket.resolvedBy)} />
        )}
      </dl>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Timeline</p>
        {detailQuery.isLoading ? (
          <Spinner className="mt-3 h-5 w-5" />
        ) : events.length === 0 ? (
          <p className="mt-3 text-sm text-text-muted">No updates yet.</p>
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
        {canManage && <FileDropZone ownerType="ticket" ownerId={ticket.id} isStaff />}
      </div>

      <div className="flex flex-col gap-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
          Linked incident
        </p>
        {ticket.incidentRef ? (
          <p className="text-sm text-text">{ticket.incidentRef}</p>
        ) : (
          <p className="text-sm text-text-muted">Not linked to an incident.</p>
        )}
        {canManage && !ticket.incidentId && (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <Select
              value={incidentToLink}
              onChange={(event) => setIncidentToLink(event.target.value)}
              aria-label="Incident to link"
              className="flex-1"
            >
              <option value="">Select an incident at this site</option>
              {siteIncidents.map((incident) => (
                <option key={incident.id} value={incident.id}>
                  {incident.ref} · {incident.title}
                </option>
              ))}
            </Select>
            <Button
              variant="secondary"
              disabled={!incidentToLink}
              onClick={() => {
                const incident = siteIncidents.find((candidate) => candidate.id === incidentToLink)
                if (!incident) return
                onRequestAction({
                  kind: 'linkIncident',
                  ticket,
                  incidentId: incident.id,
                  incidentLabel: `${incident.ref} · ${incident.title}`,
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
          For visibility only — your account doesn't hold ticket-management access.
        </p>
      )}

      {canManage && ticket.status !== 'resolved' && (
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

      {canManage && ticket.status === 'open' && (
        <Button onClick={() => onRequestAction({ kind: 'start', ticket })}>Start work</Button>
      )}

      {canManage && (ticket.status === 'open' || ticket.status === 'in_progress') && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Resolve</p>
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={resolveNote}
              onChange={(event) => setResolveNote(event.target.value)}
              placeholder="How was it resolved?"
              aria-label="Resolution note"
            />
            <Button
              disabled={!resolveNote.trim()}
              onClick={() => onRequestAction({ kind: 'resolve', ticket, note: resolveNote.trim() })}
              fullWidth
            >
              Resolve ticket
            </Button>
          </div>
        </div>
      )}

      {canManage && ticket.status === 'resolved' && (
        <div className="rounded-lg border border-border bg-surface-subtle p-4">
          {ticket.resolution && <p className="text-sm text-text">{ticket.resolution}</p>}
          <p className="mt-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
            Work it again
          </p>
          <div className="mt-3 flex flex-col gap-3">
            <Textarea
              value={reopenNote}
              onChange={(event) => setReopenNote(event.target.value)}
              placeholder="Why is this being worked again?"
              aria-label="Reason for reopening"
            />
            <Button
              variant="outline"
              disabled={!reopenNote.trim()}
              onClick={() =>
                onRequestAction({ kind: 'reopenInternal', ticket, note: reopenNote.trim() })
              }
              fullWidth
            >
              Take back into work
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

export function SupervisorTickets() {
  const { can } = usePermissions()
  const canManage = can('tickets.manage')

  const ticketsQuery = useTickets()
  const tickets = ticketsQuery.data ?? NO_TICKETS
  const summaryQuery = useTicketSummary()
  const incidentsQuery = useReporterIncidents()
  const incidents = incidentsQuery.data ?? NO_INCIDENTS

  const [tab, setTab] = useState<TabValue>('open')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null)

  const transitionMutation = useTransitionTicketMutation()
  const linkMutation = useLinkTicketIncidentMutation()
  const mutationPending = transitionMutation.isPending || linkMutation.isPending

  const counts = useMemo(() => {
    const result = { open: 0, inProgress: 0, resolved: 0 }
    for (const ticket of tickets) result[bucketFor(ticket)]++
    return result
  }, [tickets])

  const visibleTickets = useMemo(
    () => (tab === 'all' ? tickets : tickets.filter((t) => bucketFor(t) === tab)),
    [tickets, tab]
  )

  const selectedTicket =
    visibleTickets.find((t) => t.id === selectedId) ?? visibleTickets[0] ?? null
  const siteIncidents = useMemo(
    () =>
      selectedTicket ? incidents.filter((i) => i.siteId === selectedTicket.siteId) : NO_INCIDENTS,
    [incidents, selectedTicket]
  )

  const tabItems: TabItem<TabValue>[] = [
    { value: 'open', label: `Open ${counts.open}` },
    { value: 'inProgress', label: `In progress ${counts.inProgress}` },
    { value: 'resolved', label: `Resolved ${counts.resolved}` },
    { value: 'all', label: `All ${tickets.length}` },
  ]

  async function handleConfirm() {
    if (!pendingAction) return
    try {
      switch (pendingAction.kind) {
        case 'start':
          await transitionMutation.mutateAsync({
            id: pendingAction.ticket.id,
            input: { status: 'in_progress' },
          })
          toast.success('Ticket started.')
          break
        case 'resolve':
          await transitionMutation.mutateAsync({
            id: pendingAction.ticket.id,
            input: { status: 'resolved', note: pendingAction.note },
          })
          toast.success('Ticket resolved.')
          break
        case 'reopenInternal':
          await transitionMutation.mutateAsync({
            id: pendingAction.ticket.id,
            input: { status: 'in_progress', note: pendingAction.note },
          })
          toast.success('Ticket back in progress.')
          break
        case 'linkIncident':
          await linkMutation.mutateAsync({
            ticketId: pendingAction.ticket.id,
            incidentId: pendingAction.incidentId,
          })
          toast.success('Incident linked.')
          break
      }
      setPendingAction(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update this ticket.')
      setPendingAction(null)
    }
  }

  const confirmCopy: Record<PendingAction['kind'], { title: string; description: string }> = {
    start: {
      title: 'Start work on this ticket?',
      description: 'Moves it into progress and assigns it to you if nobody owns it yet.',
    },
    resolve: {
      title: 'Resolve this ticket?',
      description: 'Records the resolution below and tells Airtel it is done.',
    },
    reopenInternal: {
      title: 'Take this ticket back into work?',
      description: 'Clears the resolution and moves it back to in progress.',
    },
    linkIncident: {
      title: 'Link this incident?',
      description: 'Ties the ticket to the incident you selected.',
    },
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Support"
        title="Tickets"
        subtitle="Work the tickets Airtel raises against your sites."
      />

      {summaryQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <StatCard label="Open" value={summaryQuery.data?.open ?? 0} icon={TicketIcon} />
          <StatCard
            label="In progress"
            value={summaryQuery.data?.inProgress ?? 0}
            icon={TicketIcon}
          />
          <StatCard label="Resolved" value={summaryQuery.data?.resolved ?? 0} icon={TicketIcon} />
          <StatCard
            label="Critical open"
            value={summaryQuery.data?.criticalOpen ?? 0}
            icon={TicketIcon}
          />
        </div>
      )}

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {ticketsQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : visibleTickets.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              visibleTickets.map((ticket) => {
                const selected = ticket.id === (selectedTicket?.id ?? null)
                const { label, tone } = ticketStatusLabel(ticket.status)
                const { label: prLabel, tone: prTone } = ticketPriorityLabel(ticket.priority)
                return (
                  <li key={ticket.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(ticket.id)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs text-text-muted">{ticket.ref}</span>
                        <div className="flex gap-1.5">
                          <StatusBadge label={prLabel} tone={prTone} />
                          <StatusBadge label={label} tone={tone} />
                        </div>
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {ticket.subject} · {nameOrId(ticket.siteName, ticket.siteId)}
                      </p>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {ticketsQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedTicket ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select a ticket to see its details.
            </div>
          ) : (
            <TicketDetailBody
              key={selectedTicket.id}
              ticket={selectedTicket}
              siteLabel={nameOrId(selectedTicket.siteName, selectedTicket.siteId)}
              siteIncidents={siteIncidents}
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
        tone={pendingAction?.kind === 'resolve' ? 'success' : 'info'}
        icon={TicketIcon}
        confirmLabel="Confirm"
        pending={mutationPending}
        body={
          pendingAction && (
            <ConfirmSummary
              title={`${pendingAction.ticket.subject} · ${nameOrId(pendingAction.ticket.siteName, pendingAction.ticket.siteId)}`}
              meta={pendingAction.ticket.ref}
              rows={
                pendingAction.kind === 'linkIncident'
                  ? [{ label: 'Incident', value: pendingAction.incidentLabel }]
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
