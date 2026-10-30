import {
  useEffect,
  useId,
  useRef,
  type KeyboardEvent,
  type MouseEvent,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { XMarkIcon } from '@heroicons/react/24/outline'

import { IconButton } from './IconButton'

export type ModalMaxWidth = 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl'

export interface ModalProps {
  open: boolean
  /** Omit for a gate that can't be dismissed — no close button, Escape and
      backdrop click become no-ops (frontend/CLAUDE.md §7). */
  onClose?: () => void
  eyebrow?: string
  title: string
  subtitle?: ReactNode
  footer?: ReactNode
  maxWidth?: ModalMaxWidth
  children: ReactNode
}

const MAX_WIDTH_CLASSES: Record<ModalMaxWidth, string> = {
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
  '3xl': 'max-w-3xl',
  '4xl': 'max-w-4xl',
}

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => el.offsetParent !== null
  )
}

export function Modal({
  open,
  onClose,
  eyebrow,
  title,
  subtitle,
  footer,
  maxWidth = 'lg',
  children,
}: ModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const subtitleId = useId()

  // Focus the dialog on open (not the first field — autofocusing a <select>
  // can pop its options open in some browsers, and it would skip announcing
  // the title first). Restore focus to whatever triggered it on close.
  useEffect(() => {
    if (!open) return
    const previouslyFocused = document.activeElement as HTMLElement | null
    dialogRef.current?.focus()
    return () => {
      previouslyFocused?.focus()
    }
  }, [open])

  // Reentrant-safe scroll lock: restore the *previous* value, not '', so a
  // future ConfirmDialog stacking on top of this Modal won't prematurely
  // unlock scroll when it closes first.
  useEffect(() => {
    if (!open) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = previousOverflow
    }
  }, [open])

  if (!open) return null

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === 'Escape') {
      onClose?.()
      return
    }
    if (event.key !== 'Tab' || !dialogRef.current) return
    const focusable = getFocusable(dialogRef.current)
    if (focusable.length === 0) {
      event.preventDefault()
      return
    }
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last?.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first?.focus()
    }
  }

  function handlePanelClick(event: MouseEvent<HTMLDivElement>) {
    event.stopPropagation()
  }

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={subtitle ? subtitleId : undefined}
        tabIndex={-1}
        onClick={handlePanelClick}
        onKeyDown={handleKeyDown}
        className={`flex max-h-[calc(100dvh-2rem)] w-full flex-col rounded-lg bg-surface shadow-xl focus:outline-none ${MAX_WIDTH_CLASSES[maxWidth]}`}
      >
        <div className="flex items-start justify-between gap-4 border-b border-border p-4 sm:p-6">
          <div className="min-w-0">
            {eyebrow && (
              <p className="text-xs font-semibold uppercase tracking-wide text-primary-600">
                {eyebrow}
              </p>
            )}
            <h2 id={titleId} className="mt-1 text-lg font-bold text-text sm:text-xl">
              {title}
            </h2>
            {subtitle && (
              <p id={subtitleId} className="mt-1 text-sm text-text-muted">
                {subtitle}
              </p>
            )}
          </div>
          {onClose && (
            <IconButton
              icon={XMarkIcon}
              label="Close"
              variant="secondary"
              onClick={onClose}
              className="shrink-0 border border-border"
            />
          )}
        </div>

        <div className="min-w-0 flex-1 overflow-y-auto p-4 sm:p-6">{children}</div>

        {footer && (
          <div className="flex flex-col gap-2 border-t border-border p-4 sm:flex-row sm:items-center sm:justify-end sm:gap-3 sm:p-6">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body
  )
}
