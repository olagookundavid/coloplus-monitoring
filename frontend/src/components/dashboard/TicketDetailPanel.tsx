import { MapPinIcon, PaperClipIcon } from '@heroicons/react/24/outline'

import { useSitePerformance } from '../../hooks'
import type { Ticket } from '../../types'
import { formatDateTimeWAT, getTicketPriorityTone, getTicketStatusTone } from '../../utils'
import { SidePanel, StatusBadge } from '../ui'

export interface TicketDetailPanelProps {
  ticket: Ticket | null
  open: boolean
  onClose: () => void
}

const TICKET_DETAIL_FIELDS: Array<{
  label: string
  value: (ticket: Ticket, siteLabel: string) => string
}> = [
  { label: 'Site', value: (_ticket, siteLabel) => siteLabel },
  { label: 'Category', value: (ticket) => ticket.category },
  { label: 'Contact email', value: (ticket) => ticket.contactEmail },
  { label: 'Created', value: (ticket) => formatDateTimeWAT(ticket.createdAt) },
]

export function TicketDetailPanel({ ticket, open, onClose }: TicketDetailPanelProps) {
  // All sites, unfiltered — this panel shows whichever site the ticket
  // belongs to regardless of the page's own region/period filter.
  const sitesQuery = useSitePerformance({})
  const site = sitesQuery.data?.find((candidate) => candidate.code === ticket?.siteCode)

  if (!ticket) return null

  const siteLabel = site ? `${site.name} (${site.code}) · ${site.region}` : ticket.siteCode

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={`${ticket.id} · ${ticket.category}`}
      title={ticket.subject}
      subtitle={
        <div className="flex flex-col gap-2">
          <span className="inline-flex items-center gap-1.5">
            <MapPinIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
            {siteLabel}
          </span>
          <span className="flex items-center gap-3">
            <StatusBadge label={ticket.status} tone={getTicketStatusTone(ticket.status)} />
            <StatusBadge
              label={`${ticket.priority} priority`}
              tone={getTicketPriorityTone(ticket.priority)}
            />
          </span>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">Ticket details</h3>
          <dl className="mt-4 divide-y divide-border">
            {TICKET_DETAIL_FIELDS.map(({ label, value }) => (
              <div
                key={label}
                className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <dt className="text-sm text-text-muted">{label}</dt>
                <dd className="text-sm font-medium text-text sm:text-right">
                  {value(ticket, siteLabel)}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">What happened?</h3>
          <p className="mt-3 text-sm text-text-muted">{ticket.description}</p>
        </div>

        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">Supporting files</h3>
          {ticket.attachmentNames.length === 0 ? (
            <p className="mt-3 text-sm text-text-muted">No files attached.</p>
          ) : (
            <ul className="mt-3 flex flex-col gap-2">
              {ticket.attachmentNames.map((name) => (
                <li key={name} className="flex items-center gap-2 text-sm text-text">
                  <PaperClipIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
                  <span className="min-w-0 truncate">{name}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </SidePanel>
  )
}
