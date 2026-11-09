import { useState } from 'react'
import { MapPinIcon } from '@heroicons/react/24/outline'

import {
  useAdminSites,
  useAttachments,
  useCommentTicketMutation,
  usePermissions,
  useReopenTicketMutation,
  useTicketDetail,
} from '../../hooks'
import type { Ticket } from '../../types'
import {
  formatDateTimeWAT,
  nameOrId,
  ticketPriorityLabel,
  ticketStatusLabel,
  toast,
} from '../../utils'
import {
  AttachmentList,
  Button,
  FileDropZone,
  Input,
  SidePanel,
  Spinner,
  StatusBadge,
  Textarea,
} from '../ui'

export interface TicketDetailPanelProps {
  ticket: Ticket | null
  open: boolean
  onClose: () => void
}

function FactRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
      <dt className="text-text-muted">{label}</dt>
      <dd className="max-w-[60%] text-right font-medium text-text">{value}</dd>
    </div>
  )
}

export function TicketDetailPanel({ ticket, open, onClose }: TicketDetailPanelProps) {
  const sitesQuery = useAdminSites()
  const site = sitesQuery.data?.find((candidate) => candidate.id === ticket?.siteId)

  const detailQuery = useTicketDetail(ticket?.id)
  const events = detailQuery.data?.events ?? []
  const commentMutation = useCommentTicketMutation()
  const reopenMutation = useReopenTicketMutation()

  const { can } = usePermissions()
  const canAttach = can(['tickets.raise', 'tickets.manage'])
  const isStaff = !can('published.read')
  const attachmentsQuery = useAttachments('ticket', ticket?.id)

  const [commentBody, setCommentBody] = useState('')
  const [reopenReason, setReopenReason] = useState('')

  if (!ticket) return null

  const ticketId = ticket.id
  const siteLabel = site
    ? `${site.name} (${site.customerSiteCode}) · ${site.region}`
    : nameOrId(ticket.siteName, ticket.siteId)
  const { label: statusLabel, tone: statusTone } = ticketStatusLabel(ticket.status)
  const { label: priorityLabel, tone: priorityTone } = ticketPriorityLabel(ticket.priority)

  async function handleComment() {
    const body = commentBody.trim()
    if (!body) return
    try {
      await commentMutation.mutateAsync({ id: ticketId, body })
      setCommentBody('')
      toast.success('Comment added.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not add this comment.')
    }
  }

  async function handleReopen() {
    const note = reopenReason.trim()
    if (!note) return
    try {
      await reopenMutation.mutateAsync({ id: ticketId, note })
      setReopenReason('')
      toast.success('Ticket reopened.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not reopen this ticket.')
    }
  }

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={ticket.ref}
      title={ticket.subject}
      subtitle={
        <div className="flex flex-col gap-2">
          <span className="inline-flex items-center gap-1.5">
            <MapPinIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
            {siteLabel}
          </span>
          <span className="flex items-center gap-3">
            <StatusBadge label={statusLabel} tone={statusTone} />
            <StatusBadge label={priorityLabel} tone={priorityTone} />
          </span>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">What happened?</h3>
          <p className="mt-3 text-sm text-text-muted">{ticket.description}</p>
        </div>

        {isStaff && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Ownership</h3>
            <dl className="mt-3 divide-y divide-border rounded-lg border border-border">
              {ticket.raisedBy && (
                <FactRow label="Raised by" value={nameOrId(ticket.raisedByName, ticket.raisedBy)} />
              )}
              <FactRow
                label="Assigned to"
                value={
                  ticket.assigneeId
                    ? nameOrId(ticket.assigneeName, ticket.assigneeId)
                    : 'Unassigned'
                }
              />
              {ticket.resolvedBy && (
                <FactRow
                  label="Resolved by"
                  value={nameOrId(ticket.resolvedByName, ticket.resolvedBy)}
                />
              )}
            </dl>
          </div>
        )}

        <div className="flex flex-col gap-4 rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">Attachments</h3>
          <AttachmentList
            attachments={attachmentsQuery.data ?? []}
            loading={attachmentsQuery.isLoading}
          />
          {canAttach && <FileDropZone ownerType="ticket" ownerId={ticketId} isStaff={isStaff} />}
        </div>

        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">Updates</h3>
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
                    {event.body ||
                      `${event.party === 'customer' ? 'You' : 'Coloplus'} updated the status`}
                    {isStaff && event.authorName && (
                      <span className="text-text-muted"> · {event.authorName}</span>
                    )}
                  </span>
                </li>
              ))}
            </ul>
          )}

          <div className="mt-4 flex gap-2">
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
        </div>

        {ticket.status === 'resolved' && (
          <div className="rounded-lg border border-border bg-surface-subtle p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
              Not resolved?
            </p>
            {ticket.resolution && <p className="mt-2 text-sm text-text">{ticket.resolution}</p>}
            <div className="mt-3 flex flex-col gap-3">
              <Textarea
                value={reopenReason}
                onChange={(event) => setReopenReason(event.target.value)}
                placeholder="Why does this need to reopen?"
                aria-label="Reopen reason"
              />
              <Button
                variant="outline"
                disabled={!reopenReason.trim() || reopenMutation.isPending}
                loading={reopenMutation.isPending}
                onClick={handleReopen}
                fullWidth
              >
                Reopen ticket
              </Button>
            </div>
          </div>
        )}
      </div>
    </SidePanel>
  )
}
