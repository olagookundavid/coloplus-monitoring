import { useState } from 'react'
import { KeyIcon } from '@heroicons/react/24/outline'

import { RegenerateRecoveryCodesModal } from '../auth'
import type { User } from '../../types'
import { Button, Modal, StatusBadge } from '../ui'

export interface SecurityModalProps {
  user: User
  open: boolean
  onClose: () => void
}

export function SecurityModal({ user, open, onClose }: SecurityModalProps) {
  const [regenerateOpen, setRegenerateOpen] = useState(false)

  return (
    <>
      <Modal open={open} onClose={onClose} title="Security and 2-step verification" maxWidth="md">
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-4 py-3">
            <div>
              <p className="text-sm font-medium text-text">Two-step verification</p>
              <p className="text-sm text-text-muted">
                {user.mfaEnabled ? 'Your account is enrolled.' : 'Not required for your account.'}
              </p>
            </div>
            <StatusBadge
              label={user.mfaEnabled ? 'On' : 'Off'}
              tone={user.mfaEnabled ? 'success' : 'neutral'}
            />
          </div>

          {user.mfaEnabled && (
            <Button
              variant="outline"
              icon={KeyIcon}
              onClick={() => {
                onClose()
                setRegenerateOpen(true)
              }}
            >
              Regenerate recovery codes
            </Button>
          )}
        </div>
      </Modal>

      <RegenerateRecoveryCodesModal
        open={regenerateOpen}
        onClose={() => setRegenerateOpen(false)}
      />
    </>
  )
}
