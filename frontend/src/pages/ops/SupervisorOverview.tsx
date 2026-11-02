import { useMemo } from 'react'
import {
  BoltIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import { SkeletonListItems, SkeletonStatCards } from '../../components/skeletons'
import { ButtonLink, DataTable, StatusBadge, type Column } from '../../components/ui'
import { ROUTES } from '../../constants'
import {
  useAdminSites,
  useDashboardOverview,
  useIncidentSummary,
  usePendingApprovals,
  useReporterIncidents,
} from '../../hooks'
import type { PendingApproval } from '../../services'
import type { AdminSite, ReporterIncident } from '../../types'
import { formatAge, severityLabel } from '../../utils'

const NO_SITES: AdminSite[] = []
const NO_APPROVALS: PendingApproval[] = []
const NO_INCIDENTS: ReporterIncident[] = []

export function SupervisorOverview() {
  const sitesQuery = useAdminSites()
  const dashboardQuery = useDashboardOverview()
  const incidentsQuery = useReporterIncidents()
  const summaryQuery = useIncidentSummary()
  const approvalsQuery = usePendingApprovals()

  const sites = sitesQuery.data ?? NO_SITES
  const incidents = incidentsQuery.data ?? NO_INCIDENTS
  const approvals = approvalsQuery.data ?? NO_APPROVALS
  const sitesById = useMemo(() => new Map(sites.map((s) => [s.id, s])), [sites])

  const openIncidents = useMemo(
    () => incidents.filter((i) => i.status !== 'closed'),
    [incidents]
  )

  const incidentColumns: Column<ReporterIncident>[] = useMemo(
    () => [
      {
        header: 'Severity',
        accessor: (i) => i.severity,
        cell: (i) => {
          const { label, tone } = severityLabel(i.severity)
          return <StatusBadge label={label} tone={tone} />
        },
      },
      {
        header: 'Incident',
        accessor: (i) => `${i.title} · ${sitesById.get(i.siteId)?.airtelCode ?? i.siteId}`,
      },
      { header: 'Status', accessor: (i) => i.status.replace(/_/g, ' ') },
      { header: 'Started', accessor: (i) => formatAge(i.startedAt), align: 'right' },
    ],
    [sitesById]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Overview"
        subtitle="Incidents, pending reviews and sites across your scope."
        actions={<ButtonLink to={ROUTES.OPS_SUPERVISOR_SITES}>View sites</ButtonLink>}
      />

      {summaryQuery.isLoading || approvalsQuery.isLoading || sitesQuery.isLoading ? (
        <SkeletonStatCards count={4} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <StatCard
            label="Open incidents"
            value={summaryQuery.data?.open ?? 0}
            icon={ExclamationTriangleIcon}
            caption={`${summaryQuery.data?.critical ?? 0} critical`}
          />
          <StatCard
            label="Overdue updates"
            value={summaryQuery.data?.overdue ?? 0}
            icon={ClockIcon}
          />
          <StatCard
            label="Reports to review"
            value={approvals.length}
            icon={CheckCircleIcon}
          />
          <StatCard
            label="Outages this week"
            value={dashboardQuery.data?.outagesThisWeek ?? 0}
            icon={BoltIcon}
            caption={`${sites.length} sites in scope`}
          />
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="min-w-0 lg:col-span-2">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-base font-semibold text-text">Incidents needing attention</h2>
            <ButtonLink to={ROUTES.OPS_SUPERVISOR_INCIDENTS} variant="link" size="sm">
              View all
            </ButtonLink>
          </div>
          <DataTable
            data={openIncidents.slice(0, 6)}
            columns={incidentColumns}
            getRowId={(i) => i.id}
            loading={incidentsQuery.isLoading}
            skeletonRows={4}
            searchable={false}
            paginated={false}
            emptyMessage="No open incidents."
          />
        </div>

        <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-base font-semibold text-text">Reports to review</h2>
            <ButtonLink to={ROUTES.OPS_APPROVALS} variant="link" size="sm">
              Review
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
                    {formatAge(approval.submittedAt)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
