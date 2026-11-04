import { useState } from 'react'

import { useUpdateUserSitesMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import type { AdminSite, ManagedUser } from '../../types'
import { toast } from '../../utils'
import { Button, Checkbox, Modal } from '../ui'

export interface EditAuditorSitesModalProps {
  user: ManagedUser | null
  sites: AdminSite[]
  open: boolean
  onClose: () => void
}

// An auditor more plausibly needs many sites at once (a whole region, or
// every site) rather than the one-reporter/one-supervisor-per-site shape
// SiteAssignments.tsx offers -- a plain checklist against the full site
// list, saved with the same useUpdateUserSitesMutation that page uses.
export function EditAuditorSitesModal({ user, sites, open, onClose }: EditAuditorSitesModalProps) {
  const mutation = useUpdateUserSitesMutation()
  const [selected, setSelected] = useState<string[]>(user?.siteIds ?? [])
  const [formError, setFormError] = useState<string | null>(null)

  function resetForm() {
    setSelected(user?.siteIds ?? [])
    setFormError(null)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  function toggleSite(siteId: string, checked: boolean) {
    setSelected((current) =>
      checked ? [...current, siteId] : current.filter((id) => id !== siteId)
    )
  }

  async function handleSave() {
    if (!user) return
    setFormError(null)
    try {
      await mutation.mutateAsync({ id: user.id, siteIds: selected })
      toast.success(`Updated ${user.name}'s site access.`)
      handleClose()
    } catch (error) {
      setFormError(
        error instanceof AuthApiError ? error.message : 'Could not update their site access.'
      )
    }
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Edit sites"
      subtitle={user ? `Which sites ${user.name} can see.` : undefined}
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button onClick={() => void handleSave()} loading={mutation.isPending}>
            Save
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        <div className="flex flex-col gap-1">
          {sites.map((site) => (
            <Checkbox
              key={site.id}
              label={site.name}
              hint={`${site.airtelCode} · ${site.state}`}
              checked={selected.includes(site.id)}
              onChange={(event) => toggleSite(site.id, event.target.checked)}
            />
          ))}
        </div>
      </div>
    </Modal>
  )
}
