import { useMemo, useState } from 'react'
import { PlusIcon } from '@heroicons/react/24/outline'
import { useNavigate } from 'react-router-dom'

import { DashboardPageHeader, InviteUserModal, UserDetailPanel } from '../../components/dashboard'
import { Button, DataTable, StatusBadge, Tabs, type Column } from '../../components/ui'
import { ROUTES } from '../../constants'
import { useAdminSites, useManagedUsers } from '../../hooks'
import {
  STAFF_ROLES,
  STAFF_ROLE_LABELS,
  type AdminSite,
  type ManagedUser,
  type StaffRole,
} from '../../types'
import {
  formatDateTimeWAT,
  getInitials,
  getManagedUserStatus,
  getManagedUserStatusTone,
} from '../../utils'

type RoleFilter = StaffRole | 'all'

const NO_USERS: ManagedUser[] = []
const NO_SITES: AdminSite[] = []

export function Users() {
  const navigate = useNavigate()
  const usersQuery = useManagedUsers()
  const sitesQuery = useAdminSites()
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all')
  const [inviteOpen, setInviteOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null)

  const users = usersQuery.data ?? NO_USERS
  const sites = sitesQuery.data ?? NO_SITES
  // Derived by id, not a frozen snapshot -- so the panel reflects the
  // latest data once a mutation (role change, suspend, MFA reset)
  // invalidates and refetches the list.
  const selectedUser = users.find((user) => user.id === selectedUserId) ?? null

  const roleCounts = useMemo(() => {
    const counts = new Map<StaffRole, number>()
    for (const user of users) {
      counts.set(user.role, (counts.get(user.role) ?? 0) + 1)
    }
    return counts
  }, [users])

  const filteredUsers = useMemo(() => {
    if (roleFilter === 'all') return users
    return users.filter((user) => user.role === roleFilter)
  }, [users, roleFilter])

  const columns: Column<ManagedUser>[] = useMemo(
    () => [
      {
        header: 'User',
        accessor: (user) => user.name,
        cell: (user) => (
          <div className="flex items-center gap-3">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-soft text-xs font-semibold text-primary-700">
              {getInitials(user.name)}
            </span>
            <div className="min-w-0">
              <p className="truncate font-medium text-text">{user.name}</p>
              <p className="truncate text-xs text-text-muted">{user.email}</p>
            </div>
          </div>
        ),
      },
      {
        header: 'Role',
        accessor: (user) => STAFF_ROLE_LABELS[user.role],
      },
      {
        header: 'Site scope',
        accessor: (user) =>
          user.role === 'admin'
            ? 'All sites'
            : `${user.siteIds?.length ?? 0} site${user.siteIds?.length === 1 ? '' : 's'}`,
      },
      {
        header: 'Status',
        accessor: (user) => getManagedUserStatus(user),
        cell: (user) => {
          const status = getManagedUserStatus(user)
          return <StatusBadge label={status} tone={getManagedUserStatusTone(status)} />
        },
      },
      {
        header: 'Last active',
        accessor: (user) => user.lastLoginAt ?? '',
        cell: (user) => (user.lastLoginAt ? formatDateTimeWAT(user.lastLoginAt) : '—'),
      },
    ],
    []
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Users"
        subtitle="Manage accounts, roles and site scope. Accounts are suspended, never deleted, so history stays intact."
        actions={
          <Button icon={PlusIcon} onClick={() => setInviteOpen(true)}>
            Invite user
          </Button>
        }
      />

      <Tabs<RoleFilter>
        aria-label="Filter by role"
        items={[
          { value: 'all', label: `All (${users.length})` },
          ...STAFF_ROLES.map((role) => ({
            value: role,
            label: `${STAFF_ROLE_LABELS[role]} (${roleCounts.get(role) ?? 0})`,
          })),
        ]}
        value={roleFilter}
        onChange={setRoleFilter}
      />

      <DataTable
        data={filteredUsers}
        columns={columns}
        getRowId={(user) => user.id}
        loading={usersQuery.isLoading}
        searchPlaceholder="Search name or email..."
        emptyMessage="No users match this filter."
        onRowClick={(user) => setSelectedUserId(user.id)}
      />

      <InviteUserModal open={inviteOpen} onClose={() => setInviteOpen(false)} />

      <UserDetailPanel
        user={selectedUser}
        sites={sites}
        open={selectedUser !== null}
        onClose={() => setSelectedUserId(null)}
        onEditSiteScope={() => navigate(ROUTES.OPS_SITE_ASSIGNMENTS)}
      />
    </div>
  )
}
