import { useMemo, useState } from 'react'
import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'

import { AuditEventDetailPanel, DashboardPageHeader } from '../../components/dashboard'
import { Button, DataTable, Select, StatusBadge, Tabs, type Column } from '../../components/ui'
import { useAuditEvents, useManagedUsers } from '../../hooks'
import {
  AUDIT_CATEGORY_LABELS,
  AUDIT_CATEGORY_TONES,
  type AuditCategory,
  type AuditEvent,
  type ManagedUser,
} from '../../types'
import {
  categorizeAuditEvent as categorize,
  describeAuditEvent as describeEvent,
  downloadCsv,
  formatAuditWhen,
  formatDateTimeWAT,
  resolveAuditActorName as resolveActorName,
  toCsv,
} from '../../utils'

const NO_EVENTS: AuditEvent[] = []
const NO_USERS: ManagedUser[] = []

type DateRange = 'all' | 'today' | '7d' | '30d'
type CategoryFilter = AuditCategory | 'all'

const WAT_OFFSET_MS = 60 * 60 * 1000 // Africa/Lagos is UTC+1 year-round, no DST

function watDayStart(date: Date): Date {
  const shifted = new Date(date.getTime() + WAT_OFFSET_MS)
  shifted.setUTCHours(0, 0, 0, 0)
  return new Date(shifted.getTime() - WAT_OFFSET_MS)
}

function computeFrom(range: DateRange): string | undefined {
  const now = new Date()
  switch (range) {
    case 'today':
      return watDayStart(now).toISOString()
    case '7d':
      return new Date(now.getTime() - 7 * 24 * 3_600_000).toISOString()
    case '30d':
      return new Date(now.getTime() - 30 * 24 * 3_600_000).toISOString()
    default:
      return undefined
  }
}

export function AuditLog() {
  const usersQuery = useManagedUsers()
  const users = usersQuery.data ?? NO_USERS
  const usersById = useMemo(() => new Map(users.map((u) => [u.id, u])), [users])

  const [actorId, setActorId] = useState('')
  const [dateRange, setDateRange] = useState<DateRange>('all')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null)

  const eventsQuery = useAuditEvents({
    actorId: actorId || undefined,
    from: computeFrom(dateRange),
  })
  const events = eventsQuery.data?.events ?? NO_EVENTS

  const filteredEvents = useMemo(() => {
    if (categoryFilter === 'all') return events
    return events.filter((event) => categorize(event) === categoryFilter)
  }, [events, categoryFilter])

  const selectedEvent = filteredEvents.find((event) => event.id === selectedEventId) ?? null

  const columns: Column<AuditEvent>[] = useMemo(
    () => [
      {
        header: 'When',
        accessor: (event) => event.occurredAt,
        sortable: true,
        cell: (event) => {
          const { time, day } = formatAuditWhen(event.occurredAt)
          return (
            <div>
              <p className="font-medium text-text">{time}</p>
              <p className="text-xs text-text-muted">{day}</p>
            </div>
          )
        },
      },
      { header: 'Who', accessor: (event) => resolveActorName(event, usersById) },
      {
        header: 'Category',
        accessor: (event) => AUDIT_CATEGORY_LABELS[categorize(event)],
        cell: (event) => {
          const category = categorize(event)
          return (
            <StatusBadge
              label={AUDIT_CATEGORY_LABELS[category]}
              tone={AUDIT_CATEGORY_TONES[category]}
            />
          )
        },
      },
      {
        header: 'Action',
        accessor: (event) => describeEvent(event, categorize(event), usersById),
      },
    ],
    [usersById]
  )

  function handleExport() {
    const rows: Array<Array<string | number | null>> = [
      ['When (WAT)', 'Who', 'Category', 'Action', 'Entity type', 'Entity ID', 'Metadata'],
      ...filteredEvents.map((event) => [
        formatDateTimeWAT(event.occurredAt),
        resolveActorName(event, usersById),
        AUDIT_CATEGORY_LABELS[categorize(event)],
        describeEvent(event, categorize(event), usersById),
        event.entityType,
        event.entityId,
        JSON.stringify(event.metadata),
      ]),
    ]
    downloadCsv(`coloplus-audit-log-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows))
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Audit log"
        subtitle="Every change to accounts, assignments and approvals. Entries can't be edited or deleted."
        actions={
          <Button variant="outline" icon={ArrowDownTrayIcon} onClick={handleExport}>
            Export CSV
          </Button>
        }
      />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-4">
        <Select
          aria-label="Filter by person"
          value={actorId}
          onChange={(event) => setActorId(event.target.value)}
          className="sm:max-w-56"
        >
          <option value="">Everyone</option>
          {users.map((user) => (
            <option key={user.id} value={user.id}>
              {user.name}
            </option>
          ))}
        </Select>

        <Select
          aria-label="Filter by date"
          value={dateRange}
          onChange={(event) => setDateRange(event.target.value as DateRange)}
          className="sm:max-w-44"
        >
          <option value="all">All time</option>
          <option value="today">Today</option>
          <option value="7d">Last 7 days</option>
          <option value="30d">Last 30 days</option>
        </Select>

        <Tabs<CategoryFilter>
          aria-label="Filter by category"
          items={[
            { value: 'all', label: 'All' },
            { value: 'user', label: 'User' },
            { value: 'assignment', label: 'Assignment' },
            { value: 'approval', label: 'Approval' },
            { value: 'other', label: 'Other' },
          ]}
          value={categoryFilter}
          onChange={setCategoryFilter}
        />
      </div>

      <p className="text-xs text-text-muted">
        Export reflects the filters above, limited to the most recently loaded 200 events. Times
        shown in Africa/Lagos (WAT). The search box below searches only the loaded events, not the
        full history.
      </p>

      <DataTable
        data={filteredEvents}
        columns={columns}
        getRowId={(event) => String(event.id)}
        loading={eventsQuery.isLoading}
        searchPlaceholder="Search person or item..."
        emptyMessage="No matching audit events."
        onRowClick={(event) => setSelectedEventId(event.id)}
      />

      <AuditEventDetailPanel
        event={selectedEvent}
        description={
          selectedEvent ? describeEvent(selectedEvent, categorize(selectedEvent), usersById) : ''
        }
        category={selectedEvent ? categorize(selectedEvent) : 'other'}
        actorName={selectedEvent ? resolveActorName(selectedEvent, usersById) : ''}
        open={selectedEvent !== null}
        onClose={() => setSelectedEventId(null)}
      />
    </div>
  )
}
