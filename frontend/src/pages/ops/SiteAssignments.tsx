import { useMemo, useState } from 'react'
import { BriefcaseIcon, MapPinIcon, UserGroupIcon, XMarkIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, StatCard } from '../../components/dashboard'
import { SkeletonStatCards } from '../../components/skeletons'
import {
  Checkbox,
  DataTable,
  IconButton,
  Select,
  StatusBadge,
  Tabs,
  type Column,
} from '../../components/ui'
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

  // A site can have any number of field officers and supervisors (user_sites
  // has no per-site limit), so these return everyone assigned.
  function fieldOfficersFor(siteId: string): ManagedUser[] {
    return reporters.filter((r) => r.siteIds?.includes(siteId))
  }
  function supervisorsFor(siteId: string): ManagedUser[] {
    return supervisors.filter((s) => s.siteIds?.includes(siteId))
  }
  function needsAttention(site: AdminSite): boolean {
    return fieldOfficersFor(site.id).length === 0 || supervisorsFor(site.id).length === 0
  }

  const states = useMemo(() => [...new Set(sites.map((s) => s.state))].sort(), [sites])

  const visibleSites = useMemo(() => {
    let out = sites
    if (stateFilter !== 'all') out = out.filter((s) => s.state === stateFilter)
    if (needsAttentionOnly) out = out.filter(needsAttention)
    return out
    // eslint-disable-next-line react-hooks/exhaustive-deps -- needsAttention closes over reporters/supervisors, already deps
  }, [sites, stateFilter, needsAttentionOnly, reporters, supervisors])

  const sitesWithOfficer = sites.filter((s) => fieldOfficersFor(s.id).length > 0).length
  const statesWithoutSupervisor = states.filter(
    (state) => !sites.some((s) => s.state === state && supervisorsFor(s.id).length > 0)
  )
  const avgSitesPerOfficer = reporters.length > 0 ? sitesWithOfficer / reporters.length : 0

  // Adds or removes one person on one site. PATCH /users/:id replaces that
  // person's whole site list, so we send their list with the site added or
  // taken out; nobody else is touched.
  async function changeAssignment(
    role: 'reporter' | 'supervisor',
    site: AdminSite,
    person: ManagedUser,
    action: 'add' | 'remove'
  ) {
    const current = person.siteIds ?? []
    const siteIds =
      action === 'add' ? [...current, site.id] : current.filter((id) => id !== site.id)

    setPendingSiteId(`${role}:${site.id}`)
    try {
      await updateSitesMutation.mutateAsync({ id: person.id, siteIds })
      toast.success(
        action === 'add'
          ? `${person.name} now covers ${site.name}.`
          : `${person.name} no longer covers ${site.name}.`
      )
    } catch {
      toast.error('Could not update that assignment. Try again.')
    } finally {
      setPendingSiteId(null)
    }
  }

  // Everyone assigned to the site for one role, each removable, plus a picker
  // to add someone else. An empty site shows the warning instead.
  function renderAssignees(role: 'reporter' | 'supervisor', site: AdminSite) {
    const pool = role === 'reporter' ? reporters : supervisors
    const assigned = pool.filter((person) => person.siteIds?.includes(site.id))
    const available = pool.filter((person) => !person.siteIds?.includes(site.id))
    const noun = role === 'reporter' ? 'field officer' : 'supervisor'
    const pending = pendingSiteId === `${role}:${site.id}`

    return (
      <div className="flex min-w-48 flex-col gap-1">
        {assigned.length === 0 ? (
          <StatusBadge
            label={role === 'reporter' ? 'Unassigned' : 'Needs supervisor'}
            tone="warning"
          />
        ) : (
          assigned.map((person) => (
            <div key={person.id} className="flex items-center justify-between gap-2">
              <span className="min-w-0 truncate text-sm text-text">{person.name}</span>
              <IconButton
                icon={XMarkIcon}
                label={`Remove ${person.name} from ${site.name}`}
                disabled={pending}
                onClick={() => void changeAssignment(role, site, person, 'remove')}
              />
            </div>
          ))
        )}
        {available.length > 0 && (
          <Select
            value=""
            disabled={pending}
            onChange={(event) => {
              const person = available.find((p) => p.id === event.target.value)
              if (person) void changeAssignment(role, site, person, 'add')
            }}
            aria-label={`Add a ${noun} to ${site.name}`}
            className="w-full"
          >
            <option value="">
              {assigned.length === 0 ? `Add a ${noun}` : `Add another ${noun}`}
            </option>
            {available.map((person) => (
              <option key={person.id} value={person.id}>
                {person.name}
              </option>
            ))}
          </Select>
        )}
      </div>
    )
  }

  const columns: Column<AdminSite>[] = useMemo(
    () => [
      { header: 'Site', accessor: (site) => site.customerSiteCode },
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
        header: 'Field officers',
        accessor: (site) =>
          fieldOfficersFor(site.id)
            .map((person) => person.name)
            .join(', ') || 'Unassigned',
        cell: (site) => renderAssignees('reporter', site),
      },
      {
        header: 'Supervisors',
        accessor: (site) =>
          supervisorsFor(site.id)
            .map((person) => person.name)
            .join(', ') || 'Needs supervisor',
        cell: (site) => renderAssignees('supervisor', site),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps -- cells close over reporters/supervisors/pendingSiteId, already deps
    [reporters, supervisors, pendingSiteId]
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Site assignments"
        subtitle="Decide who reports on each site and who supervises it. A site can have several of each. Reporters and supervisors only see sites assigned to them; reviewers, viewers and auditors get their sites from the Users page."
      />

      {sitesQuery.isLoading || usersQuery.isLoading ? (
        <SkeletonStatCards count={3} />
      ) : (
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
      )}

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
