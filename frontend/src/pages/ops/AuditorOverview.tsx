import { useMemo } from 'react'
import {
  ClipboardDocumentCheckIcon,
  MapIcon,
  WrenchScrewdriverIcon,
} from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import { SkeletonListItems, SkeletonStatCards } from '../../components/skeletons'
import { ButtonLink } from '../../components/ui'
import { ROUTES } from '../../constants'
import {
  useAdminSites,
  useManagedUsers,
  useMaintenanceWindows,
  usePendingApprovals,
  usePermissions,
  useRecentAuditEvents,
} from '../../hooks'
import type { PendingApproval } from '../../services'
import type { AdminSite, ManagedUser } from '../../types'
import { formatAge, formatDateTimeWAT } from '../../utils'

const NO_SITES: AdminSite[] = []
const NO_APPROVALS: PendingApproval[] = []
const NO_USERS: ManagedUser[] = []

// useRecentAuditEvents returns the lighter AuditEvent shape (no metadata),
// so it uses this local action map rather than the richer utils/
// auditDisplay.ts helpers AuditLog.tsx uses against the fuller useAuditEvents
// shape -- same split pages/ops/Overview.tsx's own preview card already has.
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

export function AuditorOverview() {
  const { can } = usePermissions()

  const sitesQuery = useAdminSites()
  const approvalsQuery = usePendingApprovals()
  const maintenanceQuery = useMaintenanceWindows({ status: 'requested' })
  const auditQuery = useRecentAuditEvents(8)
  // Degrades to raw actor/author ids when disabled -- auditor lacks
  // users.manage, so this mirrors AuditLog.tsx's own conditional fetch.
  const usersQuery = useManagedUsers(can('users.manage'))

  const sites = sitesQuery.data ?? NO_SITES
  const approvals = approvalsQuery.data ?? NO_APPROVALS
  const users = usersQuery.data ?? NO_USERS
  const sitesById = useMemo(() => new Map(sites.map((s) => [s.id, s])), [sites])
  const usersById = useMemo(() => new Map(users.map((u) => [u.id, u])), [users])

  const pendingMaintenance = maintenanceQuery.data?.windows ?? []

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Overview"
        subtitle="Portfolio reports and audit history across your assigned sites."
        actions={<ButtonLink to={ROUTES.OPS_AUDIT_LOG}>View audit log</ButtonLink>}
      />

      {sitesQuery.isLoading || approvalsQuery.isLoading || maintenanceQuery.isLoading ? (
        <SkeletonStatCards count={3} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <StatCard label="Sites in scope" value={sites.length} icon={MapIcon} />
          <StatCard
            label="Reports awaiting review"
            value={approvals.length}
            icon={ClipboardDocumentCheckIcon}
            caption={`${countOlderThan48Hours(approvals)} older than 48 hours`}
          />
          <StatCard
            label="Maintenance requests pending"
            value={pendingMaintenance.length}
            icon={WrenchScrewdriverIcon}
          />
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="min-w-0 rounded-lg border border-border bg-surface p-5 lg:col-span-2">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-base font-semibold text-text">Reports awaiting review</h2>
            <ButtonLink to={ROUTES.OPS_APPROVALS} variant="link" size="sm">
              Open
            </ButtonLink>
          </div>
          {approvalsQuery.isLoading ? (
            <SkeletonListItems count={3} withBadge={false} />
          ) : approvals.length === 0 ? (
            <p className="text-sm text-text-muted">Nothing waiting on review.</p>
          ) : (
            <ul className="flex flex-col gap-3">
              {approvals.slice(0, 5).map((approval) => (
                <li key={approval.id} className="text-sm">
                  <p className="font-medium text-text">
                    {sitesById.get(approval.siteId)?.name ?? approval.siteId} ·{' '}
                    {approval.reportDate}
                  </p>
                  <p className="text-xs text-text-muted">
                    {approval.isCorrection ? 'Correction' : 'Publication'} ·{' '}
                    {usersById.get(approval.authorId)?.name ?? approval.authorId} ·{' '}
                    {formatAge(approval.submittedAt)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
          <h2 className="mb-3 text-base font-semibold text-text">Recent audit activity</h2>
          {auditQuery.isLoading ? (
            <ul className="flex flex-col gap-3">
              <SkeletonListItems count={3} withBadge={false} />
            </ul>
          ) : (auditQuery.data?.length ?? 0) === 0 ? (
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
  )
}
