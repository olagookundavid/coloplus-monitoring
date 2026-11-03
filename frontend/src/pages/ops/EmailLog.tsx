import { useState } from 'react'
import { ArrowPathIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { DataTable, RowActions, StatusBadge, Tabs, type Column } from '../../components/ui'
import { useNotifications, useRetryNotificationMutation } from '../../hooks'
import type { Notification, NotificationStatus } from '../../types'
import {
  formatAuditWhen,
  humanizeNotificationKind,
  notificationStatusLabel,
  toast,
} from '../../utils'

const NO_NOTIFICATIONS: Notification[] = []

type StatusFilter = NotificationStatus | 'all'

export function EmailLog() {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')

  const notificationsQuery = useNotifications({
    status: statusFilter === 'all' ? undefined : statusFilter,
    limit: 200,
  })
  const notifications = notificationsQuery.data?.notifications ?? NO_NOTIFICATIONS

  const retryMutation = useRetryNotificationMutation()

  async function handleRetry(notification: Notification) {
    try {
      await retryMutation.mutateAsync(notification.id)
      toast.success('Email queued to send again.')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not retry this email.')
    }
  }

  // Not useMemo'd: the Actions column's RowActions closures need the
  // current render's retryMutation/handleRetry, matching the fuel entry
  // history table's own RowActions column (components/dashboard/FuelPanel.tsx).
  const columns: Column<Notification>[] = [
    {
      header: 'When',
      accessor: (notification) => notification.createdAt,
      sortable: true,
      cell: (notification) => {
        const { time, day } = formatAuditWhen(notification.createdAt)
        return (
          <div>
            <p className="font-medium text-text">{time}</p>
            <p className="text-xs text-text-muted">{day}</p>
          </div>
        )
      },
    },
    {
      header: 'Kind',
      accessor: (notification) => humanizeNotificationKind(notification.kind),
    },
    { header: 'Recipient', accessor: (notification) => notification.recipient },
    {
      header: 'Subject',
      accessor: (notification) => notification.subject,
      cell: (notification) => <span className="line-clamp-2">{notification.subject}</span>,
    },
    {
      header: 'Status',
      accessor: (notification) => notificationStatusLabel(notification.status).label,
      cell: (notification) => {
        const { label, tone } = notificationStatusLabel(notification.status)
        return <StatusBadge label={label} tone={tone} />
      },
    },
    {
      header: 'Attempts',
      accessor: (notification) => notification.attempts,
      align: 'right',
    },
    {
      header: 'Last error',
      accessor: (notification) => notification.lastError ?? null,
      cell: (notification) => (
        <span className="text-xs text-text-muted">{notification.lastError || '—'}</span>
      ),
    },
    {
      header: 'Actions',
      accessor: () => null,
      align: 'right',
      cell: (notification) => (
        <RowActions
          label="Email actions"
          actions={[
            {
              label: 'Retry',
              icon: ArrowPathIcon,
              disabled: notification.status !== 'failed',
              hint:
                notification.status !== 'failed' ? 'Only a failed email can be retried' : undefined,
              onClick: () => void handleRetry(notification),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Email log"
        subtitle="Delivery status of system emails. The message text is never shown here."
      />

      <Tabs<StatusFilter>
        aria-label="Filter by status"
        items={[
          { value: 'all', label: 'All' },
          { value: 'pending', label: 'Pending' },
          { value: 'sent', label: 'Sent' },
          { value: 'failed', label: 'Failed' },
        ]}
        value={statusFilter}
        onChange={setStatusFilter}
      />

      <p className="text-xs text-text-muted">
        Showing the most recent 200 matching emails. Times shown in Africa/Lagos (WAT). The search
        box below searches only the loaded emails, not the full history.
      </p>

      <DataTable
        data={notifications}
        columns={columns}
        getRowId={(notification) => notification.id}
        loading={notificationsQuery.isLoading}
        searchPlaceholder="Search recipient or subject..."
        emptyMessage="No emails match this filter."
      />
    </div>
  )
}
