import { useState } from 'react'

import { useUpdateUserSitesMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import type { AdminSite, ManagedUser } from '../../types'
import { toast } from '../../utils'
import { Button, Modal } from '../ui'
import { SiteChecklist } from './SiteChecklist'

export interface EditUserSitesModalProps {
  user: ManagedUser | null
  sites: AdminSite[]
  open: boolean
  onClose: () => void
}

// Which sites one person can see: a checklist against the full site list,
// for any role that is limited to sites (supervisor, reporter, reviewer,
// viewer, auditor). Saved with the same useUpdateUserSitesMutation that
// Site assignments uses.
export function EditUserSitesModal({ user, sites, open, onClose }: EditUserSitesModalProps) {
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

        <SiteChecklist sites={sites} selected={selected} onChange={setSelected} />
      </div>
    </Modal>
  )
}
