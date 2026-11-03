import { useEffect, useMemo, useState } from 'react'
import { CheckCircleIcon, ClockIcon, LifebuoyIcon, PlusIcon, TicketIcon } from '@heroicons/react/24/outline'
import { useNavigate, useParams } from 'react-router-dom'

import { DashboardPageHeader, StatCard, TicketDetailPanel } from '../../components/dashboard'
import { SkeletonStatCards } from '../../components/skeletons'
import { Button, DataTable, Select, StatusBadge, type Column } from '../../components/ui'
import { ROUTES, ticketDetailPath } from '../../constants'
import { useAdminSites, useTicketSummary, useTickets } from '../../hooks'
import type { AdminSite, Ticket, TicketStatus } from '../../types'
import { formatDateTimeWAT, ticketPriorityLabel, ticketStatusLabel } from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const NO_TICKETS: Ticket[] = []
const NO_SITES: AdminSite[] = []

const STATUS_FILTER_OPTIONS: Array<TicketStatus | 'all'> = ['all', 'open', 'in_progress', 'resolved']

export function SupportTickets() {
  const { openCreateTicket } = useDashboardFilters()
  const [statusFilter, setStatusFilter] = useState<TicketStatus | 'all'>('all')
  const { id: selectedTicketId } = useParams<{ id?: string }>()
  const navigate = useNavigate()

  const ticketsQuery = useTickets()
  const tickets = ticketsQuery.data ?? NO_TICKETS
  const summaryQuery = useTicketSummary()
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.id, site])), [sites])

  const selectedTicket = selectedTicketId
    ? (tickets.find((ticket) => ticket.id === selectedTicketId) ?? null)
    : null

  // A stale or mistyped ticket id in the URL — send the customer back to the
  // list rather than leaving the panel open with nothing to show.
  useEffect(() => {
    if (selectedTicketId && !ticketsQuery.isLoading && !selectedTicket) {
      navigate(ROUTES.CUSTOMER_TICKETS, { replace: true })
    }
  }, [selectedTicketId, ticketsQuery.isLoading, selectedTicket, navigate])

  const filteredTickets = useMemo(() => {
    if (statusFilter === 'all') return tickets
    return tickets.filter((ticket) => ticket.status === statusFilter)
  }, [tickets, statusFilter])

  const columns: Column<Ticket>[] = useMemo(
    () => [
      {
        header: 'Ticket / Subject',
        accessor: (ticket) => ticket.subject,
        cell: (ticket) => (
          <div>
            <p className="text-xs font-medium text-primary-600">{ticket.ref}</p>
            <p className="font-medium text-text">{ticket.subject}</p>
          </div>
        ),
        sortable: true,
      },
      {
        header: 'Site',
        accessor: (ticket) => sitesById.get(ticket.siteId)?.airtelCode ?? ticket.siteId,
        sortable: true,
      },
      {
        header: 'Priority',
        accessor: (ticket) => ticket.priority,
        cell: (ticket) => {
          const { label, tone } = ticketPriorityLabel(ticket.priority)
          return <StatusBadge label={label} tone={tone} />
        },
        sortable: true,
      },
      {
        header: 'Status',
        accessor: (ticket) => ticket.status,
        cell: (ticket) => {
          const { label, tone } = ticketStatusLabel(ticket.status)
          return <StatusBadge label={label} tone={tone} />
        },
        sortable: true,
      },
      {
        header: 'Created WAT',
        accessor: (ticket) => ticket.createdAt,
        cell: (ticket) => formatDateTimeWAT(ticket.createdAt),
        sortable: true,
      },
    ],
    [sitesById]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Customer support"
        title="Support tickets"
        subtitle="One conversation. Clear ownership. Every issue tracked."
        actions={
          <Button icon={PlusIcon} onClick={() => openCreateTicket()}>
            Create ticket
          </Button>
        }
      />

      {summaryQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard label="All tickets" value={tickets.length} icon={TicketIcon} />
          <StatCard label="Open" value={summaryQuery.data?.open ?? 0} icon={ClockIcon} />
          <StatCard label="In progress" value={summaryQuery.data?.inProgress ?? 0} icon={LifebuoyIcon} />
          <StatCard label="Resolved" value={summaryQuery.data?.resolved ?? 0} icon={CheckCircleIcon} />
        </div>
      )}

      <div>
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-text">Your support requests</h2>
          <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-text-muted">
            {filteredTickets.length}
          </span>
        </div>
        <DataTable
          data={filteredTickets}
          columns={columns}
          getRowId={(ticket) => ticket.id}
          loading={ticketsQuery.isLoading}
          getRowHref={(ticket) => ticketDetailPath(ticket.id)}
          searchable={false}
          emptyMessage="No tickets match this filter."
          headerActions={
            <Select
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value as TicketStatus | 'all')}
              aria-label="Filter by status"
              className="w-44"
            >
              {STATUS_FILTER_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option === 'all' ? 'All tickets' : ticketStatusLabel(option).label}
                </option>
              ))}
            </Select>
          }
        />
        <div className="mt-3 flex justify-end">
          <Button variant="link" size="sm" icon={PlusIcon} onClick={() => openCreateTicket()}>
            New request
          </Button>
        </div>
      </div>

      <TicketDetailPanel
        ticket={selectedTicket}
        open={Boolean(selectedTicket)}
        onClose={() => navigate(ROUTES.CUSTOMER_TICKETS)}
      />
    </div>
  )
}
