import { useState } from 'react'

import { useResetUserMfaMutation, useUpdateUserMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import {
  STAFF_ROLES,
  STAFF_ROLE_LABELS,
  type AdminSite,
  type ManagedUser,
  type StaffRole,
} from '../../types'
import { formatDateTimeWAT, getInitials, getManagedUserStatus, toast } from '../../utils'
import { Button, Modal, Select, SidePanel, StatusBadge } from '../ui'

export interface UserDetailPanelProps {
  user: ManagedUser | null
  sites: AdminSite[]
  open: boolean
  onClose: () => void
  onEditSiteScope: () => void
}

function organisationFor(role: StaffRole): string {
  return role === 'airtel_viewer' || role === 'airtel_reviewer' ? 'Airtel' : 'Coloplus'
}

function describeSiteScope(user: ManagedUser, sites: AdminSite[]): string {
  if (user.role === 'admin') return 'All sites'
  const ids = user.siteIds ?? []
  if (ids.length === 0) return 'No sites assigned'
  const states = [
    ...new Set(
      ids
        .map((id) => sites.find((site) => site.id === id)?.state)
        .filter((state): state is string => Boolean(state))
    ),
  ]
  return `${states.join(', ')} · ${ids.length} site${ids.length === 1 ? '' : 's'}`
}

export function UserDetailPanel({
  user,
  sites,
  open,
  onClose,
  onEditSiteScope,
}: UserDetailPanelProps) {
  const updateMutation = useUpdateUserMutation()
  const resetMfaMutation = useResetUserMfaMutation()
  const [confirmSuspend, setConfirmSuspend] = useState(false)

  if (!user) return null

  const status = getManagedUserStatus(user)

  async function handleRoleChange(role: StaffRole) {
    if (!user || role === user.role) return
    try {
      // Admins can never hold a site assignment (the backend rejects it) --
      // clear it in the same request rather than leaving a stale list
      // behind that the backend just never had a reason to reject.
      await updateMutation.mutateAsync({
        id: user.id,
        patch: role === 'admin' ? { role, siteIds: [] } : { role },
      })
      toast.success(`${user.name}'s role is now ${STAFF_ROLE_LABELS[role]}.`)
    } catch (error) {
      toast.error(error instanceof AuthApiError ? error.message : 'Could not change that role.')
    }
  }

  async function handleResetMfa() {
    if (!user) return
    try {
      await resetMfaMutation.mutateAsync(user.id)
      toast.success(`Two-step verification reset for ${user.name}.`)
    } catch (error) {
      toast.error(
        error instanceof AuthApiError
          ? error.message
          : "Could not reset that person's 2-step login."
      )
    }
  }

  async function confirmSuspendAccount() {
    if (!user) return
    try {
      await updateMutation.mutateAsync({ id: user.id, patch: { isActive: false } })
      toast.success(`${user.name}'s account is suspended.`)
      setConfirmSuspend(false)
    } catch (error) {
      toast.error(error instanceof AuthApiError ? error.message : 'Could not suspend that account.')
    }
  }

  async function handleReactivate() {
    if (!user) return
    try {
      await updateMutation.mutateAsync({ id: user.id, patch: { isActive: true } })
      toast.success(`${user.name}'s account is active again.`)
    } catch (error) {
      toast.error(
        error instanceof AuthApiError ? error.message : 'Could not reactivate that account.'
      )
    }
  }

  return (
    <>
      <SidePanel
        open={open}
        onClose={onClose}
        maxWidth="md"
        title={
          <div className="flex items-center gap-3">
            <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-primary-soft text-sm font-semibold text-primary-700">
              {getInitials(user.name)}
            </span>
            <span>{user.name}</span>
          </div>
        }
        subtitle={<span className="pl-14">{user.email}</span>}
        footer={
          <>
            <Button variant="outline" onClick={handleResetMfa} loading={resetMfaMutation.isPending}>
              Reset 2-step login
            </Button>
            {status === 'Suspended' ? (
              <Button
                variant="outline"
                onClick={handleReactivate}
                loading={updateMutation.isPending}
              >
                Reactivate account
              </Button>
            ) : (
              <Button variant="danger" onClick={() => setConfirmSuspend(true)}>
                Suspend account
              </Button>
            )}
          </>
        }
      >
        <div className="flex flex-col gap-6">
          <dl className="divide-y divide-border rounded-lg border border-border">
            <div className="flex items-center justify-between gap-3 px-4 py-3">
              <dt className="text-sm text-text-muted">Organisation</dt>
              <dd className="text-sm font-medium text-text">{organisationFor(user.role)}</dd>
            </div>
            <div className="flex items-center justify-between gap-3 px-4 py-3">
              <dt className="text-sm text-text-muted">Status</dt>
              <dd>
                <StatusBadge
                  label={status}
                  tone={
                    status === 'Active' ? 'success' : status === 'Invited' ? 'warning' : 'neutral'
                  }
                />
              </dd>
            </div>
            <div className="flex items-center justify-between gap-3 px-4 py-3">
              <dt className="text-sm text-text-muted">2-step login</dt>
              <dd className="text-sm font-medium text-text">{user.mfaEnabled ? 'On' : 'Off'}</dd>
            </div>
            <div className="flex items-center justify-between gap-3 px-4 py-3">
              <dt className="text-sm text-text-muted">Last active</dt>
              <dd className="text-sm font-medium text-text">
                {user.lastLoginAt ? formatDateTimeWAT(user.lastLoginAt) : 'Never'}
              </dd>
            </div>
          </dl>

          <div>
            <label htmlFor="user-panel-role" className="text-sm font-medium text-text">
              Role
            </label>
            <Select
              id="user-panel-role"
              className="mt-1.5"
              value={user.role}
              disabled={updateMutation.isPending}
              onChange={(event) => void handleRoleChange(event.target.value as StaffRole)}
            >
              {STAFF_ROLES.map((role) => (
                <option key={role} value={role}>
                  {STAFF_ROLE_LABELS[role]}
                </option>
              ))}
            </Select>
          </div>

          <div>
            <p className="text-sm font-medium text-text">Site scope</p>
            <div className="mt-1.5 flex items-center justify-between gap-3 rounded-lg border border-border px-4 py-3">
              <span className="text-sm text-text">{describeSiteScope(user, sites)}</span>
              {user.role !== 'admin' && (
                <Button variant="link" size="sm" onClick={onEditSiteScope}>
                  Edit
                </Button>
              )}
            </div>
          </div>

          <p className="rounded-lg bg-surface-subtle p-4 text-xs text-text-muted">
            Role and site changes are written to the audit log. Approving reports requires a
            separate supervisor account — administrators manage access but don't review entries.
          </p>
        </div>
      </SidePanel>

      <Modal
        open={confirmSuspend}
        onClose={() => setConfirmSuspend(false)}
        title="Suspend this account?"
        subtitle={`${user.name} won't be able to sign in. Accounts are suspended, never deleted, so their history stays intact.`}
        footer={
          <>
            <Button variant="outline" onClick={() => setConfirmSuspend(false)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={updateMutation.isPending}
              onClick={() => void confirmSuspendAccount()}
            >
              Suspend account
            </Button>
          </>
        }
      >
        <p className="text-sm text-text-muted">This takes effect immediately.</p>
      </Modal>
    </>
  )
}
