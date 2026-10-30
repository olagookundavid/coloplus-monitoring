import { useEffect, useMemo, useState } from 'react'
import {
  ArrowDownTrayIcon,
  ArrowTrendingUpIcon,
  CheckCircleIcon,
  ClockIcon,
  LifebuoyIcon,
  PlusIcon,
  TicketIcon,
} from '@heroicons/react/24/outline'
import { useNavigate, useParams } from 'react-router-dom'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  StatCard,
  TicketDetailPanel,
} from '../../components/dashboard'
import { Button, DataTable, Select, StatusBadge, type Column } from '../../components/ui'
import { ROUTES, ticketDetailPath } from '../../constants'
import { useTickets } from '../../hooks'
import type { Ticket, TicketStatus } from '../../types'
import { formatDateTimeWAT, getTicketPriorityTone, getTicketStatusTone } from '../../utils'
import { useDashboardFilters } from './CustomerLayout'

const NO_TICKETS: Ticket[] = []

const STATUS_FILTER_OPTIONS: Array<TicketStatus | 'All tickets'> = [
  'All tickets',
  'Open',
  'In progress',
  'Resolved',
]

const TICKET_COLUMNS: Column<Ticket>[] = [
  {
    header: 'Ticket / Subject',
    accessor: (ticket) => ticket.subject,
    cell: (ticket) => (
      <div>
        <p className="text-xs font-medium text-primary-600">{ticket.id}</p>
        <p className="font-medium text-text">{ticket.subject}</p>
      </div>
    ),
    sortable: true,
  },
  {
    header: 'Site',
    accessor: (ticket) => ticket.siteCode,
    sortable: true,
  },
  {
    header: 'Priority',
    accessor: (ticket) => ticket.priority,
    cell: (ticket) => (
      <StatusBadge label={ticket.priority} tone={getTicketPriorityTone(ticket.priority)} />
    ),
    sortable: true,
  },
  {
    header: 'Status',
    accessor: (ticket) => ticket.status,
    cell: (ticket) => (
      <StatusBadge label={ticket.status} tone={getTicketStatusTone(ticket.status)} />
    ),
    sortable: true,
  },
  {
    header: 'Created WAT',
    accessor: (ticket) => ticket.createdAt,
    cell: (ticket) => formatDateTimeWAT(ticket.createdAt),
    sortable: true,
  },
]

export function SupportTickets() {
  const { region, setRegion, period, setPeriod, openCreateTicket } = useDashboardFilters()
  const [statusFilter, setStatusFilter] = useState<TicketStatus | 'All tickets'>('All tickets')
  const { id: selectedTicketId } = useParams<{ id?: string }>()
  const navigate = useNavigate()

  const ticketsQuery = useTickets()
  const tickets = ticketsQuery.data ?? NO_TICKETS
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
    if (statusFilter === 'All tickets') return tickets
    return tickets.filter((ticket) => ticket.status === statusFilter)
  }, [tickets, statusFilter])

  const stats = useMemo(
    () => ({
      all: tickets.length,
      awaitingResponse: tickets.filter((ticket) => ticket.status === 'Open').length,
      inProgress: tickets.filter((ticket) => ticket.status === 'In progress').length,
      resolved: tickets.filter((ticket) => ticket.status === 'Resolved').length,
    }),
    [tickets]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Customer support"
        title="Support tickets"
        subtitle="One conversation. Clear ownership. Every issue tracked."
        actions={
          <>
            <Button variant="outline" icon={ArrowDownTrayIcon}>
              Export report
            </Button>
            <Button icon={PlusIcon} onClick={openCreateTicket}>
              Create ticket
            </Button>
          </>
        }
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        period={period}
        onPeriodChange={setPeriod}
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="All tickets"
          value={stats.all}
          icon={TicketIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: 'In the selected reporting scope',
            tone: 'positive',
          }}
        />
        <StatCard
          label="Awaiting response"
          value={stats.awaitingResponse}
          icon={ClockIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: 'Received by the support queue',
            tone: 'positive',
          }}
        />
        <StatCard
          label="In progress"
          value={stats.inProgress}
          icon={LifebuoyIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: 'Assigned to operations',
            tone: 'positive',
          }}
        />
        <StatCard
          label="Resolved"
          value={stats.resolved}
          icon={CheckCircleIcon}
          trend={{
            icon: ArrowTrendingUpIcon,
            label: 'Resolution recorded',
            tone: 'positive',
          }}
        />
      </div>

      <div>
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-text">Your support requests</h2>
          <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-text-muted">
            {filteredTickets.length}
          </span>
        </div>
        <DataTable
          data={filteredTickets}
          columns={TICKET_COLUMNS}
          getRowId={(ticket) => ticket.id}
          loading={ticketsQuery.isLoading}
          getRowHref={(ticket) => ticketDetailPath(ticket.id)}
          searchable={false}
          emptyMessage="No tickets match this filter."
          headerActions={
            <Select
              value={statusFilter}
              onChange={(event) =>
                setStatusFilter(event.target.value as TicketStatus | 'All tickets')
              }
              aria-label="Filter by status"
              className="w-44"
            >
              {STATUS_FILTER_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </Select>
          }
        />
        <div className="mt-3 flex flex-col gap-2 text-xs text-text-muted sm:flex-row sm:items-center sm:justify-between">
          <span>Demo tickets are stored in this browser. They are not sent to Coloplus.</span>
          <Button variant="link" size="sm" icon={PlusIcon} onClick={openCreateTicket}>
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
