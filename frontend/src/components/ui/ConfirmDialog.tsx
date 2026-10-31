import {
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
  type SVGProps,
} from 'react'

import { Button, type ButtonVariant } from './Button'
import { Checkbox } from './Checkbox'
import { Modal, type ModalMaxWidth } from './Modal'

export type ConfirmDialogTone = 'success' | 'info' | 'warning' | 'danger'

export interface ConfirmDialogProps {
  open: boolean
  title: string
  description?: ReactNode
  body?: ReactNode
  confirmLabel: string
  tone: ConfirmDialogTone
  icon?: ComponentType<SVGProps<SVGSVGElement>>
  pending?: boolean
  maxWidth?: ModalMaxWidth
  /** A stable, page+operation-scoped key (e.g. "ops-approvals:approve").
      Offers a "don't ask me again" checkbox; once checked and confirmed,
      later opens for this key skip straight to onConfirm. Opt-in per call
      site, and never offered when tone="danger" -- enforced here, not by
      convention, so an irreversible action can never be silenced even by a
      careless call site. */
  dontAskAgainKey?: string
  dontAskAgainLabel?: string
  onConfirm: () => void
  onCancel: () => void
}

const TONE_ICON_CLASSES: Record<ConfirmDialogTone, string> = {
  success: 'bg-primary-soft text-primary-700',
  info: 'bg-sky-50 text-sky-700',
  warning: 'bg-warning-soft text-warning-700',
  danger: 'bg-danger-50 text-danger-600',
}

const TONE_BUTTON_VARIANT: Record<ConfirmDialogTone, ButtonVariant> = {
  success: 'success',
  info: 'info',
  warning: 'warning',
  danger: 'danger',
}

function skipStorageKey(key: string): string {
  return `confirm-skip:${key}`
}

function readSkip(key: string): boolean {
  try {
    return localStorage.getItem(skipStorageKey(key)) === 'true'
  } catch {
    return false
  }
}

function writeSkip(key: string): void {
  try {
    localStorage.setItem(skipStorageKey(key), 'true')
  } catch {
    // Private browsing or storage disabled -- the checkbox just won't
    // persist across sessions, which is a fine fallback.
  }
}

export function ConfirmDialog({
  open,
  title,
  description,
  body,
  confirmLabel,
  tone,
  icon: Icon,
  pending = false,
  maxWidth = 'md',
  dontAskAgainKey,
  dontAskAgainLabel = "Don't ask me again",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const offerSkip = Boolean(dontAskAgainKey) && tone !== 'danger'
  const skipActive = offerSkip && dontAskAgainKey ? readSkip(dontAskAgainKey) : false
  const [skipFuture, setSkipFuture] = useState(false)
  const autoConfirmed = useRef(false)

  // A key with "don't ask again" already set skips the dialog body entirely
  // -- confirm fires as soon as the caller opens it, once per open session
  // (the ref guards against firing again if onConfirm's identity changes
  // while still open).
  useEffect(() => {
    if (!open) {
      autoConfirmed.current = false
      return
    }
    if (skipActive && !autoConfirmed.current) {
      autoConfirmed.current = true
      onConfirm()
    }
  }, [open, skipActive, onConfirm])

  if (skipActive) return null

  function handleConfirm() {
    if (offerSkip && skipFuture && dontAskAgainKey) {
      writeSkip(dontAskAgainKey)
    }
    onConfirm()
  }

  return (
    <Modal
      open={open}
      onClose={pending ? undefined : onCancel}
      title={title}
      subtitle={description}
      maxWidth={maxWidth}
      footer={
        <>
          <Button variant="outline" onClick={onCancel} disabled={pending}>
            Cancel
          </Button>
          <Button variant={TONE_BUTTON_VARIANT[tone]} onClick={handleConfirm} loading={pending}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {Icon && (
          <span
            className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${TONE_ICON_CLASSES[tone]}`}
          >
            <Icon className="h-5 w-5" aria-hidden="true" />
          </span>
        )}
        {body}
        {offerSkip && (
          <Checkbox
            label={dontAskAgainLabel}
            checked={skipFuture}
            onChange={(event) => setSkipFuture(event.target.checked)}
          />
        )}
      </div>
    </Modal>
  )
}
