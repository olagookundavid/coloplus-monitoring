import { useMemo, useState } from 'react'
import { BriefcaseIcon, MapPinIcon, UserGroupIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import { Checkbox, DataTable, Select, StatusBadge, Tabs, type Column } from '../../components/ui'
import { useAdminSites, useManagedUsers, useUpdateUserSitesMutation } from '../../hooks'
import type { AdminSite, ManagedUser } from '../../types'
import { toast } from '../../utils'

const NO_SITES: AdminSite[] = []
const NO_USERS: ManagedUser[] = []

export function SiteAssignments() {
  const sitesQuery = useAdminSites()
  const usersQuery = useManagedUsers()
  const updateSitesMutation = useUpdateUserSitesMutation()

  const [stateFilter, setStateFilter] = useState('all')
  const [needsAttentionOnly, setNeedsAttentionOnly] = useState(false)
  const [pendingSiteId, setPendingSiteId] = useState<string | null>(null)

  const sites = sitesQuery.data ?? NO_SITES
  const users = usersQuery.data ?? NO_USERS
  // isActive only -- a suspended account keeps its site_ids on the backend
  // (nothing clears them), but shouldn't be offered as, or counted as, an
  // assignable field officer or supervisor here.
  const reporters = useMemo(() => users.filter((u) => u.role === 'reporter' && u.isActive), [users])
  const supervisors = useMemo(
    () => users.filter((u) => u.role === 'supervisor' && u.isActive),
    [users]
  )

  function fieldOfficerFor(siteId: string): ManagedUser | undefined {
    return reporters.find((r) => r.siteIds?.includes(siteId))
  }
  function supervisorFor(siteId: string): ManagedUser | undefined {
    return supervisors.find((s) => s.siteIds?.includes(siteId))
  }
  function needsAttention(site: AdminSite): boolean {
    return !fieldOfficerFor(site.id) || !supervisorFor(site.id)
  }

  const states = useMemo(() => [...new Set(sites.map((s) => s.state))].sort(), [sites])

  const visibleSites = useMemo(() => {
    let out = sites
    if (stateFilter !== 'all') out = out.filter((s) => s.state === stateFilter)
    if (needsAttentionOnly) out = out.filter(needsAttention)
    return out
    // eslint-disable-next-line react-hooks/exhaustive-deps -- needsAttention closes over reporters/supervisors, already deps
  }, [sites, stateFilter, needsAttentionOnly, reporters, supervisors])

  const sitesWithOfficer = sites.filter((s) => fieldOfficerFor(s.id)).length
  const statesWithoutSupervisor = states.filter(
    (state) => !sites.some((s) => s.state === state && supervisorFor(s.id))
  )
  const avgSitesPerOfficer = reporters.length > 0 ? sitesWithOfficer / reporters.length : 0

  // The one field officer per site shown here is a UI convention, not a
  // schema rule (user_sites has no per-site cardinality limit) -- picking a
  // new one removes the site from whoever currently has it and adds it to
  // the new pick, so the single-select stays an accurate "who's assigned."
  async function handleFieldOfficerChange(site: AdminSite, newOfficerId: string) {
    const current = fieldOfficerFor(site.id)
    if ((current?.id ?? '') === newOfficerId) return

    setPendingSiteId(site.id)
    try {
      const tasks: Promise<unknown>[] = []
      if (current) {
        tasks.push(
          updateSitesMutation.mutateAsync({
            id: current.id,
            siteIds: (current.siteIds ?? []).filter((id) => id !== site.id),
          })
        )
      }
      if (newOfficerId) {
        const next = reporters.find((r) => r.id === newOfficerId)
        if (next) {
          tasks.push(
            updateSitesMutation.mutateAsync({
              id: next.id,
              siteIds: [...(next.siteIds ?? []), site.id],
            })
          )
        }
      }
      await Promise.all(tasks)
      toast.success(`Updated the field officer for ${site.name}.`)
    } catch {
      toast.error('Could not update that assignment. Try again.')
    } finally {
      setPendingSiteId(null)
    }
  }

  const columns: Column<AdminSite>[] = useMemo(
    () => [
      { header: 'Site', accessor: (site) => site.airtelCode },
      {
        header: 'Name and type',
        accessor: (site) => site.name,
        cell: (site) => (
          <div>
            <p className="font-medium text-text">{site.name}</p>
            <p className="text-xs text-text-muted">{site.siteType}</p>
          </div>
        ),
      },
      { header: 'State', accessor: (site) => site.state },
      {
        header: 'Field officer',
        accessor: (site) => fieldOfficerFor(site.id)?.name ?? 'Unassigned',
        cell: (site) => {
          const current = fieldOfficerFor(site.id)
          return (
            <Select
              value={current?.id ?? ''}
              disabled={pendingSiteId === site.id}
              onChange={(event) => void handleFieldOfficerChange(site, event.target.value)}
              aria-label={`Field officer for ${site.name}`}
              className="w-full min-w-40"
            >
              <option value="">Unassigned</option>
              {reporters.map((reporter) => (
                <option key={reporter.id} value={reporter.id}>
                  {reporter.name}
                </option>
              ))}
            </Select>
          )
        },
      },
      {
        header: 'Supervisor',
        accessor: (site) => supervisorFor(site.id)?.name ?? 'Needs supervisor',
        cell: (site) => {
          const supervisor = supervisorFor(site.id)
          return supervisor ? (
            supervisor.name
          ) : (
            <StatusBadge label="Needs supervisor" tone="warning" />
          )
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps -- cells close over reporters/supervisors/pendingSiteId, already deps
    [reporters, supervisors, pendingSiteId]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Site assignments"
        subtitle="Decide who reports on each site and who supervises it. Reporters only see sites assigned to them."
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          label="Sites with a field officer"
          value={`${sitesWithOfficer} / ${sites.length}`}
          icon={UserGroupIcon}
          caption={`${sites.length - sitesWithOfficer} unassigned`}
        />
        <StatCard
          label="States without a supervisor"
          value={statesWithoutSupervisor.length}
          icon={MapPinIcon}
          caption={
            statesWithoutSupervisor.length > 0
              ? `${statesWithoutSupervisor.join(', ')} need one`
              : 'Every state has one'
          }
        />
        <StatCard
          label="Field officers"
          value={reporters.length}
          icon={BriefcaseIcon}
          caption={`Avg ${avgSitesPerOfficer.toFixed(1)} sites each`}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs
          aria-label="Filter by state"
          items={[
            { value: 'all', label: 'All' },
            ...states.map((state) => ({ value: state, label: state })),
          ]}
          value={stateFilter}
          onChange={setStateFilter}
        />
        <Checkbox
          label="Needs attention only"
          checked={needsAttentionOnly}
          onChange={(event) => setNeedsAttentionOnly(event.target.checked)}
        />
      </div>

      <DataTable
        data={visibleSites}
        columns={columns}
        getRowId={(site) => site.id}
        loading={sitesQuery.isLoading || usersQuery.isLoading}
        searchPlaceholder="Search site code or name..."
        emptyMessage="No sites match this filter."
        rowClassName={(site) => (needsAttention(site) ? 'bg-warning-50' : '')}
      />
    </div>
  )
}
