import { useMemo, useState } from 'react'
import { CheckCircleIcon, ClockIcon, UserGroupIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import { ButtonLink, DataTable, StatusBadge, Tabs, type Column } from '../../components/ui'
import { ROUTES } from '../../constants'
import {
  useAdminSites,
  useManagedUsers,
  usePendingApprovals,
  useRecentAuditEvents,
} from '../../hooks'
import type { PendingApproval } from '../../services'
import type { AdminSite, ManagedUser } from '../../types'
import { STAFF_ROLE_LABELS } from '../../types'
import {
  formatAge,
  formatDateTimeWAT,
  getManagedUserStatus,
  getManagedUserStatusTone,
} from '../../utils'

const NO_USERS: ManagedUser[] = []
const NO_SITES: AdminSite[] = []
const NO_APPROVALS: PendingApproval[] = []

const ACTION_LABELS: Record<string, string> = {
  'user.created': 'created a user account',
  'user.updated': 'updated a user account',
  'user.invite_resent': 'resent an invite',
  'auth.login': 'signed in',
  'auth.mfa_enabled': 'turned on two-step verification',
  'auth.mfa_reset': 'reset two-step verification',
  'report.correction_started': 'started a report correction',
  'report.approved': 'approved a report',
  'report.rejected': 'returned a report',
  'site.created': 'created a site',
  'site.updated': 'updated a site',
  'site.activated': 'activated a site',
  'incident.reported': 'reported an incident',
}

function describeAction(action: string): string {
  return ACTION_LABELS[action] ?? action
}

function countOlderThan48Hours(approvals: PendingApproval[]): number {
  return approvals.filter(
    (a) => a.submittedAt && Date.now() - new Date(a.submittedAt).getTime() > 48 * 3_600_000
  ).length
}

type ApprovalFilter = 'all' | 'publication' | 'correction'

export function Overview() {
  const usersQuery = useManagedUsers()
  const sitesQuery = useAdminSites()
  const approvalsQuery = usePendingApprovals()
  const auditQuery = useRecentAuditEvents(8)

  const [approvalFilter, setApprovalFilter] = useState<ApprovalFilter>('all')

  const users = usersQuery.data ?? NO_USERS
  const sites = sitesQuery.data ?? NO_SITES
  const approvals = approvalsQuery.data ?? NO_APPROVALS

  const usersById = useMemo(() => new Map(users.map((u) => [u.id, u])), [users])
  const sitesById = useMemo(() => new Map(sites.map((s) => [s.id, s])), [sites])

  const activeUsers = users.filter((u) => getManagedUserStatus(u) === 'Active').length
  const invitedUsers = users.filter((u) => getManagedUserStatus(u) === 'Invited').length

  // isActive only -- matches SiteAssignments.tsx: a suspended account keeps
  // its site_ids on the backend, but shouldn't count as a real assignment.
  const reporters = useMemo(() => users.filter((u) => u.role === 'reporter' && u.isActive), [users])
  const sitesWithReporter = sites.filter((site) =>
    reporters.some((r) => r.siteIds?.includes(site.id))
  ).length
  const unassignedSites = useMemo(
    () => sites.filter((site) => !reporters.some((r) => r.siteIds?.includes(site.id))),
    [sites, reporters]
  )

  const oldApprovals = countOlderThan48Hours(approvals)

  const filteredApprovals = useMemo(() => {
    if (approvalFilter === 'all') return approvals
    if (approvalFilter === 'correction') return approvals.filter((a) => a.isCorrection)
    return approvals.filter((a) => !a.isCorrection)
  }, [approvals, approvalFilter])

  const approvalColumns: Column<PendingApproval>[] = useMemo(
    () => [
      {
        header: 'Type',
        accessor: (a) => (a.isCorrection ? 'Correction' : 'Publication'),
        cell: (a) => (
          <StatusBadge
            label={a.isCorrection ? 'Correction' : 'Publication'}
            tone={a.isCorrection ? 'warning' : 'info'}
          />
        ),
      },
      {
        header: 'Item',
        accessor: (a) => `${sitesById.get(a.siteId)?.name ?? a.siteId} · ${a.reportDate}`,
      },
      {
        header: 'Requested by',
        accessor: (a) => usersById.get(a.authorId)?.name ?? a.authorId,
      },
      { header: 'Age', accessor: (a) => formatAge(a.submittedAt), align: 'right' },
    ],
    [sitesById, usersById]
  )

  const previewUserColumns: Column<ManagedUser>[] = useMemo(
    () => [
      {
        header: 'User',
        accessor: (u) => u.name,
        cell: (u) => (
          <div>
            <p className="font-medium text-text">{u.name}</p>
            <p className="text-xs text-text-muted">{u.email}</p>
          </div>
        ),
      },
      { header: 'Role', accessor: (u) => STAFF_ROLE_LABELS[u.role] },
      {
        header: 'Status',
        accessor: (u) => getManagedUserStatus(u),
        cell: (u) => {
          const status = getManagedUserStatus(u)
          return <StatusBadge label={status} tone={getManagedUserStatusTone(status)} />
        },
      },
    ],
    []
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Administration overview"
        subtitle="Users, site assignments and configuration."
        actions={<ButtonLink to={ROUTES.OPS_USERS}>Manage users</ButtonLink>}
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          label="Active users"
          value={activeUsers}
          icon={UserGroupIcon}
          caption={`${invitedUsers} invitation${invitedUsers === 1 ? '' : 's'} pending`}
        />
        <StatCard
          label="Sites assigned"
          value={`${sitesWithReporter} / ${sites.length}`}
          icon={CheckCircleIcon}
          caption={`${sites.length - sitesWithReporter} need a reporter`}
        />
        <StatCard
          label="Approvals waiting"
          value={approvals.length}
          icon={ClockIcon}
          caption={`${oldApprovals} older than 48 hours`}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="min-w-0 lg:col-span-2">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-base font-semibold text-text">Approvals waiting</h2>
            <Tabs<ApprovalFilter>
              aria-label="Filter approvals"
              items={[
                { value: 'all', label: 'All' },
                { value: 'publication', label: 'Publication' },
                { value: 'correction', label: 'Correction' },
              ]}
              value={approvalFilter}
              onChange={setApprovalFilter}
            />
          </div>
          {/* Read-only for admin: administrators manage users and sites but
              hold no reports.review permission -- only a supervisor can
              actually approve, by design (backend/internal/sql/migrations/
              00002_identity.sql). */}
          <p className="mb-2 text-xs text-text-muted">
            For visibility only — administrators don't hold report-review access.
          </p>
          <DataTable
            data={filteredApprovals}
            columns={approvalColumns}
            getRowId={(a) => a.id}
            loading={approvalsQuery.isLoading}
            searchable={false}
            paginated={false}
            emptyMessage="Nothing waiting on approval."
          />
        </div>

        <div className="flex flex-col gap-4">
          <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-base font-semibold text-text">Unassigned sites</h2>
              <ButtonLink to={ROUTES.OPS_SITE_ASSIGNMENTS} variant="link" size="sm">
                Assign sites
              </ButtonLink>
            </div>
            {unassignedSites.length === 0 ? (
              <p className="text-sm text-text-muted">Every site has a field officer.</p>
            ) : (
              <ul className="flex flex-col gap-3">
                {unassignedSites.slice(0, 5).map((site) => (
                  <li key={site.id} className="flex items-center justify-between gap-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-text">{site.name}</p>
                      <p className="truncate text-xs text-text-muted">
                        {site.airtelCode}, {site.state}
                      </p>
                    </div>
                    <ButtonLink
                      to={ROUTES.OPS_SITE_ASSIGNMENTS}
                      variant="link"
                      size="sm"
                      className="shrink-0"
                    >
                      Assign reporter
                    </ButtonLink>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
            <h2 className="mb-3 text-base font-semibold text-text">Recent audit activity</h2>
            {(auditQuery.data?.length ?? 0) === 0 ? (
              <p className="text-sm text-text-muted">No recent activity.</p>
            ) : (
              <ul className="flex flex-col gap-3">
                {auditQuery.data?.map((event) => (
                  <li key={event.id} className="text-sm">
                    <span className="font-medium text-text">
                      {usersById.get(event.actorId ?? '')?.name ?? 'System'}
                    </span>{' '}
                    <span className="text-text-muted">{describeAction(event.action)}</span>
                    <p className="text-xs text-text-muted">{formatDateTimeWAT(event.occurredAt)}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-base font-semibold text-text">Users and access</h2>
          <ButtonLink to={ROUTES.OPS_USERS} variant="link" size="sm">
            Manage all {users.length} users
          </ButtonLink>
        </div>
        <DataTable
          data={users.slice(0, 6)}
          columns={previewUserColumns}
          getRowId={(u) => u.id}
          loading={usersQuery.isLoading}
          searchable={false}
          paginated={false}
          emptyMessage="No users yet."
        />
      </div>
    </div>
  )
}
