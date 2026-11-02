import { useMemo, useState } from 'react'
import { BellIcon } from '@heroicons/react/24/outline'
import { Link } from 'react-router-dom'

import { ROUTES } from '../../constants'
import {
  useAdminSites,
  useAuditEvents,
  useDropdownMenu,
  useManagedUsers,
  usePendingApprovals,
  usePermissions,
} from '../../hooks'
import type { PendingApproval } from '../../services'
import type { AdminSite, AuditEvent, ManagedUser } from '../../types'
import {
  categorizeAuditEvent,
  describeAuditEvent,
  formatRelativeShort,
  resolveAuditActorName,
} from '../../utils'
import { Tabs } from '../ui'

interface NotificationItem {
  id: string
  title: string
  timeLabel: string
  initials: string
  colorClass: string
  unread: boolean
  linkTo?: string
}

type FeedFilter = 'all' | 'needs-action'

const LAST_SEEN_KEY = 'coloplus.notifications.lastSeenAt'

function readLastSeen(): string {
  try {
    return localStorage.getItem(LAST_SEEN_KEY) ?? ''
  } catch {
    return ''
  }
}

function writeLastSeen(iso: string): void {
  try {
    localStorage.setItem(LAST_SEEN_KEY, iso)
  } catch {
    // Private browsing or storage disabled -- unread state just won't
    // persist across sessions, which is a fine fallback.
  }
}

const NO_EVENTS: AuditEvent[] = []
const NO_SITES: AdminSite[] = []
const NO_USERS: ManagedUser[] = []
const NO_APPROVALS: PendingApproval[] = []
const CATEGORY_STYLE: Record<string, { initials: string; colorClass: string }> = {
  approval: { initials: 'AP', colorClass: 'bg-primary-soft text-primary-700' },
  assignment: { initials: 'AS', colorClass: 'bg-warning-soft text-warning-700' },
  user: { initials: 'US', colorClass: 'bg-sky-100 text-sky-700' },
  other: { initials: 'SY', colorClass: 'bg-slate-100 text-slate-600' },
}

function useOpsNotificationItems(lastSeen: string): {
  items: NotificationItem[]
  isLoading: boolean
} {
  const usersQuery = useManagedUsers()
  const sitesQuery = useAdminSites()
  const approvalsQuery = usePendingApprovals()
  const eventsQuery = useAuditEvents({ limit: 20 })

  const users = usersQuery.data ?? NO_USERS
  const sites = sitesQuery.data ?? NO_SITES
  const approvals = approvalsQuery.data ?? NO_APPROVALS
  const events = eventsQuery.data?.events ?? NO_EVENTS
  // Captured once at mount via a lazy initializer, not read directly during
  // render -- the React Compiler's purity rule refuses any direct Date.now()
  // call in a component body (same reasoning that forbids it inside the memo
  // below). A little staleness over a long session is harmless for a
  // 48-hour threshold check.
  const [now] = useState(() => Date.now())

  const items = useMemo(() => {
    const usersById = new Map(users.map((u) => [u.id, u]))
    const result: NotificationItem[] = []

    // Computed alerts first -- a current condition, not a point-in-time
    // event, so no real timestamp and no unread state (never counted
    // toward the bell badge). Same computations Overview.tsx's stat cards
    // already use, reimplemented here rather than refactoring that page.
    const reporters = users.filter((u) => u.role === 'reporter' && u.isActive)
    const unassignedCount = sites.filter(
      (site) => !reporters.some((r) => r.siteIds?.includes(site.id))
    ).length
    if (unassignedCount > 0) {
      result.push({
        id: 'alert-unassigned-sites',
        title: `${unassignedCount} site${unassignedCount === 1 ? '' : 's'} still ${unassignedCount === 1 ? 'has' : 'have'} no field officer assigned`,
        timeLabel: 'Current',
        initials: 'ST',
        colorClass: 'bg-warning-soft text-warning-700',
        unread: false,
        linkTo: ROUTES.OPS_SITE_ASSIGNMENTS,
      })
    }

    const staleCount = approvals.filter(
      (a) => a.submittedAt && now - new Date(a.submittedAt).getTime() > 48 * 3_600_000
    ).length
    if (staleCount > 0) {
      result.push({
        id: 'alert-stale-approvals',
        title: `${staleCount} approval${staleCount === 1 ? '' : 's'} ${staleCount === 1 ? 'has' : 'have'} been waiting over 48 hours`,
        timeLabel: 'Current',
        initials: 'AP',
        colorClass: 'bg-primary-soft text-primary-700',
        unread: false,
        linkTo: ROUTES.OPS_APPROVALS,
      })
    }

    for (const event of events) {
      const category = categorizeAuditEvent(event)
      const style = CATEGORY_STYLE[category] ?? CATEGORY_STYLE.other!
      const description = describeAuditEvent(event, category, usersById)
      const lowered = description.charAt(0).toLowerCase() + description.slice(1)
      result.push({
        id: `event-${event.id}`,
        title: `${resolveAuditActorName(event, usersById)} ${lowered}`,
        timeLabel: formatRelativeShort(event.occurredAt),
        initials: style.initials,
        colorClass: style.colorClass,
        // Never visited before -> everything currently loaded counts as new.
        unread: !lastSeen || event.occurredAt > lastSeen,
        linkTo: category === 'approval' ? ROUTES.OPS_APPROVALS : undefined,
      })
    }

    return result
  }, [users, sites, approvals, events, lastSeen, now])

  return { items, isLoading: eventsQuery.isLoading }
}

interface OpsNotificationsPanelProps {
  lastSeen: string
  onMarkAllRead: () => void
  onNavigate: () => void
}

// Only ever mounted for an audit.read holder (admin today) -- its hooks are
// never called, and so never fire, for anyone else. Keeps the Topbar (shared
// by both the ops and customer portals) from issuing admin-only queries for
// a customer-portal user.
function OpsNotificationsPanel({
  lastSeen,
  onMarkAllRead,
  onNavigate,
}: OpsNotificationsPanelProps) {
  const [filter, setFilter] = useState<FeedFilter>('all')
  const { items, isLoading } = useOpsNotificationItems(lastSeen)
  const unreadCount = items.filter((item) => item.unread).length
  const visibleItems = filter === 'needs-action' ? items.filter((item) => item.linkTo) : items

  return (
    <>
      <div className="flex items-center justify-between gap-3 border-b border-border p-4">
        <h2 className="text-sm font-semibold text-text">Notifications</h2>
        {unreadCount > 0 && (
          <button
            type="button"
            onClick={onMarkAllRead}
            className="text-sm font-medium text-primary hover:underline"
          >
            Mark all as read
          </button>
        )}
      </div>

      <div className="border-b border-border px-4 pt-3">
        <Tabs<FeedFilter>
          aria-label="Filter notifications"
          items={[
            { value: 'all', label: 'All' },
            { value: 'needs-action', label: 'Needs action' },
          ]}
          value={filter}
          onChange={setFilter}
        />
      </div>

      <ul className="min-h-0 flex-1 overflow-y-auto">
        {isLoading ? (
          <li className="p-6 text-center text-sm text-text-muted">Loading…</li>
        ) : visibleItems.length === 0 ? (
          <li className="p-6 text-center text-sm text-text-muted">Nothing here.</li>
        ) : (
          visibleItems.map((item) => {
            const row = (
              <div
                className={`flex items-start gap-3 px-4 py-3 ${item.unread ? 'bg-primary-soft/40' : ''}`}
              >
                <span
                  className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold ${item.colorClass}`}
                >
                  {item.initials}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-text">{item.title}</p>
                  <p className="mt-0.5 text-xs text-text-muted">{item.timeLabel}</p>
                </div>
                {item.unread && (
                  <span
                    className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary"
                    aria-hidden="true"
                  />
                )}
              </div>
            )
            return (
              <li key={item.id} className="border-b border-border last:border-b-0">
                {item.linkTo ? (
                  <Link
                    to={item.linkTo}
                    onClick={onNavigate}
                    className="block transition-colors hover:bg-surface-subtle"
                  >
                    {row}
                  </Link>
                ) : (
                  row
                )}
              </li>
            )
          })
        )}
      </ul>

      <div className="border-t border-border p-3 text-center">
        <Link
          to={ROUTES.OPS_AUDIT_LOG}
          onClick={onNavigate}
          className="text-sm font-medium text-primary hover:underline"
        >
          View all activity
        </Link>
      </div>
    </>
  )
}

// Mounted whenever hasAccess is true, independent of whether the dropdown
// is open, so the bell shows a count even before it's clicked. Shares a
// TanStack Query cache (same hooks, same query keys) with
// OpsNotificationsPanel below rather than lifting state -- when both are
// mounted together (dropdown open), the second subscriber reuses the
// already-fetched data instead of firing a second request.
function NotificationsBadge({ lastSeen }: { lastSeen: string }) {
  const { items } = useOpsNotificationItems(lastSeen)
  const unreadCount = items.filter((item) => item.unread).length
  if (unreadCount === 0) return null
  return (
    <span className="absolute right-1.5 top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-[10px] font-semibold text-white">
      {unreadCount > 9 ? '9+' : unreadCount}
    </span>
  )
}

export function NotificationsMenu() {
  const { can } = usePermissions()
  const hasAccess = can('audit.read')
  const { open, setOpen, containerRef } = useDropdownMenu<HTMLDivElement>()
  const [lastSeen, setLastSeen] = useState(readLastSeen)

  function markAllRead() {
    const now = new Date().toISOString()
    writeLastSeen(now)
    setLastSeen(now)
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Notifications"
        className="relative flex h-11 w-11 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text"
      >
        <BellIcon className="h-5 w-5" aria-hidden="true" />
        {hasAccess && <NotificationsBadge lastSeen={lastSeen} />}
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-2 flex max-h-[28rem] w-80 flex-col overflow-hidden rounded-lg border border-border bg-surface shadow-lg sm:w-96"
        >
          {hasAccess ? (
            <OpsNotificationsPanel
              lastSeen={lastSeen}
              onMarkAllRead={markAllRead}
              onNavigate={() => setOpen(false)}
            />
          ) : (
            <>
              <div className="border-b border-border p-4">
                <h2 className="text-sm font-semibold text-text">Notifications</h2>
              </div>
              <p className="p-6 text-center text-sm text-text-muted">No notifications yet.</p>
            </>
          )}
        </div>
      )}
    </div>
  )
}
