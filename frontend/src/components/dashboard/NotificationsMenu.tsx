import { useState } from 'react'
import { BellIcon } from '@heroicons/react/24/outline'

import {
  useAcknowledgeAlertMutation,
  useAcknowledgeAllAlertsMutation,
  useAlerts,
  useDropdownMenu,
  usePermissions,
} from '../../hooks'
import type { Alert } from '../../types'
import { formatRelativeShort, toast } from '../../utils'
import { Tabs } from '../ui'

const NO_ALERTS: Alert[] = []

type AlertFeedFilter = 'unread' | 'all'

// Mounted independently of whether the dropdown is open, so the bell shows
// a count even before it's clicked. Its query params match
// AlertCentrePanel's own initial ("Unread") tab exactly, so opening the
// panel reuses this already-fetched cache entry instead of firing a second
// request (same TanStack Query cache/key, per useAlerts.ts's own comment).
function AlertCentreBadge() {
  const { data } = useAlerts({ unread: true, limit: 20 })
  const unread = data?.unread ?? 0
  if (unread === 0) return null
  return (
    <span className="absolute right-1.5 top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-[10px] font-semibold text-white">
      {unread > 9 ? '9+' : unread}
    </span>
  )
}

function AlertCentrePanel() {
  const [filter, setFilter] = useState<AlertFeedFilter>('unread')
  const alertsQuery = useAlerts({ unread: filter === 'unread' ? true : undefined, limit: 20 })
  const alerts = alertsQuery.data?.alerts ?? NO_ALERTS
  const unread = alertsQuery.data?.unread ?? 0

  const acknowledgeMutation = useAcknowledgeAlertMutation()
  const acknowledgeAllMutation = useAcknowledgeAllAlertsMutation()

  async function handleAcknowledge(id: string) {
    try {
      await acknowledgeMutation.mutateAsync(id)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not acknowledge this alert.')
    }
  }

  async function handleAcknowledgeAll() {
    try {
      const result = await acknowledgeAllMutation.mutateAsync()
      toast.success(
        `Acknowledged ${result.acknowledged} alert${result.acknowledged === 1 ? '' : 's'}.`
      )
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not acknowledge your alerts.')
    }
  }

  return (
    <>
      <div className="flex items-center justify-between gap-3 border-b border-border p-4">
        <h2 className="text-sm font-semibold text-text">Alerts</h2>
        {unread > 0 && (
          <button
            type="button"
            onClick={() => void handleAcknowledgeAll()}
            className="text-sm font-medium text-primary hover:underline"
          >
            Acknowledge all
          </button>
        )}
      </div>

      <div className="border-b border-border px-4 pt-3">
        <Tabs<AlertFeedFilter>
          aria-label="Filter alerts"
          items={[
            { value: 'unread', label: 'Unread' },
            { value: 'all', label: 'All' },
          ]}
          value={filter}
          onChange={setFilter}
        />
      </div>

      <ul className="min-h-0 flex-1 overflow-y-auto">
        {alertsQuery.isLoading ? (
          <li className="p-6 text-center text-sm text-text-muted">Loading…</li>
        ) : alerts.length === 0 ? (
          <li className="p-6 text-center text-sm text-text-muted">Nothing here.</li>
        ) : (
          alerts.map((alert) => {
            const isUnread = !alert.acknowledgedAt
            const row = (
              <div
                className={`flex items-start gap-3 px-4 py-3 text-left ${isUnread ? 'bg-primary-soft/40' : ''}`}
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-text">{alert.title}</p>
                  {alert.body && (
                    <p className="mt-0.5 line-clamp-2 text-xs text-text-muted">{alert.body}</p>
                  )}
                  <p className="mt-0.5 text-xs text-text-muted">
                    {formatRelativeShort(alert.createdAt)}
                  </p>
                </div>
                {isUnread && (
                  <span
                    className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary"
                    aria-hidden="true"
                  />
                )}
              </div>
            )
            return (
              <li key={alert.id} className="border-b border-border last:border-b-0">
                {isUnread ? (
                  <button
                    type="button"
                    onClick={() => void handleAcknowledge(alert.id)}
                    className="block w-full transition-colors hover:bg-surface-subtle"
                  >
                    {row}
                  </button>
                ) : (
                  row
                )}
              </li>
            )
          })
        )}
      </ul>
    </>
  )
}

// alerts.read is granted to every role (own alerts only) -- this guard is
// "authorise by permission, never role name" defensiveness, not a real
// restriction today.
export function NotificationsMenu() {
  const { can } = usePermissions()
  const hasAccess = can('alerts.read')
  const { open, setOpen, containerRef } = useDropdownMenu<HTMLDivElement>()

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Alerts"
        className="relative flex h-11 w-11 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text"
      >
        <BellIcon className="h-5 w-5" aria-hidden="true" />
        {hasAccess && <AlertCentreBadge />}
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-2 flex max-h-[28rem] w-80 flex-col overflow-hidden rounded-lg border border-border bg-surface shadow-lg sm:w-96"
        >
          {hasAccess ? (
            <AlertCentrePanel />
          ) : (
            <>
              <div className="border-b border-border p-4">
                <h2 className="text-sm font-semibold text-text">Alerts</h2>
              </div>
              <p className="p-6 text-center text-sm text-text-muted">No alerts yet.</p>
            </>
          )}
        </div>
      )}
    </div>
  )
}
