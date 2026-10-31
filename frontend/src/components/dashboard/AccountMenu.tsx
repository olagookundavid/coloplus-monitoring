import { useState } from 'react'
import {
  ArrowRightOnRectangleIcon,
  ChevronDownIcon,
  ShieldCheckIcon,
  UserCircleIcon,
} from '@heroicons/react/24/outline'

import { useAuth } from '../../contexts'
import { useDropdownMenu } from '../../hooks'
import { STAFF_ROLE_LABELS, type StaffRole } from '../../types'
import { ConfirmDialog } from '../ui'
import { MyProfileModal } from './MyProfileModal'
import { SecurityModal } from './SecurityModal'

function getInitials(name: string | undefined): string {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean)
  const first = parts[0]?.[0] ?? ''
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : ''
  return `${first}${last}`.toUpperCase() || '?'
}

export function AccountMenu() {
  const { user, logout } = useAuth()
  const { open, setOpen, containerRef } = useDropdownMenu<HTMLDivElement>()
  const [profileOpen, setProfileOpen] = useState(false)
  const [securityOpen, setSecurityOpen] = useState(false)
  const [confirmSignOut, setConfirmSignOut] = useState(false)
  const [signingOut, setSigningOut] = useState(false)

  if (!user) return null

  const roleLabel = STAFF_ROLE_LABELS[user.role as StaffRole] ?? user.role

  async function handleSignOut() {
    setSigningOut(true)
    await logout()
    setSigningOut(false)
    setConfirmSignOut(false)
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex items-center gap-2 rounded-full border border-border py-1 pl-1 pr-2.5 text-sm transition-colors hover:bg-surface-subtle"
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-soft text-xs font-semibold text-primary-700">
          {getInitials(user.name)}
        </span>
        <span className="hidden max-w-32 truncate font-medium text-text sm:inline">
          {user.name}
        </span>
        <ChevronDownIcon
          className={`h-4 w-4 shrink-0 text-text-muted transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden="true"
        />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-2 w-64 overflow-hidden rounded-lg border border-border bg-surface shadow-lg"
        >
          <div className="p-4">
            <p className="truncate text-sm font-semibold text-text">{user.name}</p>
            <p className="truncate text-xs text-text-muted">{roleLabel}</p>
          </div>
          <div className="border-t border-border p-1.5">
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                setProfileOpen(true)
              }}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-text transition-colors hover:bg-surface-subtle"
            >
              <UserCircleIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
              My profile
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                setSecurityOpen(true)
              }}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-text transition-colors hover:bg-surface-subtle"
            >
              <ShieldCheckIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
              Security and 2-step verification
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                setConfirmSignOut(true)
              }}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-danger-600 transition-colors hover:bg-danger-50"
            >
              <ArrowRightOnRectangleIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
              Sign out
            </button>
          </div>
        </div>
      )}

      <MyProfileModal user={user} open={profileOpen} onClose={() => setProfileOpen(false)} />
      <SecurityModal user={user} open={securityOpen} onClose={() => setSecurityOpen(false)} />

      <ConfirmDialog
        open={confirmSignOut}
        title="Log out?"
        confirmLabel="Log out"
        tone="info"
        icon={ArrowRightOnRectangleIcon}
        pending={signingOut}
        onConfirm={() => void handleSignOut()}
        onCancel={() => setConfirmSignOut(false)}
      />
    </div>
  )
}
