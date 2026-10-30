import { useEffect, useId, useRef, useState, type MouseEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { XMarkIcon } from '@heroicons/react/24/outline'

import { useDialogLifecycle } from './dialogBehavior'
import { IconButton } from './IconButton'
import type { ModalMaxWidth } from './Modal'

export interface SidePanelProps {
  open: boolean
  onClose?: () => void
  eyebrow?: string
  title: string
  /** Caller composes whatever meta row it needs (a location line, status
      badges) — same philosophy as Modal's subtitle. */
  subtitle?: ReactNode
  /** Slot beside the close button, e.g. a "Create ticket" action. */
  headerActions?: ReactNode
  maxWidth?: ModalMaxWidth
  children: ReactNode
}

const MAX_WIDTH_CLASSES: Record<ModalMaxWidth, string> = {
  md: 'sm:max-w-md',
  lg: 'sm:max-w-lg',
  xl: 'sm:max-w-xl',
  '2xl': 'sm:max-w-2xl',
  '3xl': 'sm:max-w-3xl',
  '4xl': 'sm:max-w-4xl',
}

export function SidePanel({
  open,
  onClose,
  eyebrow,
  title,
  subtitle,
  headerActions,
  maxWidth = 'xl',
  children,
}: SidePanelProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const subtitleId = useId()
  const { handleKeyDown } = useDialogLifecycle(open, dialogRef, onClose)

  // Mount translated fully off-screen, then flip one frame later so the
  // transform transition actually has something to animate from — setting
  // both classes in the same render paints the open state immediately.
  // `entered` resets when `open` flips to false by comparing against a
  // previous-value state (React's own "adjusting state during render"
  // pattern — a ref can't be read/written during render under this
  // project's React Compiler lint rules), so a same-instance reopen
  // animates again instead of staying stuck from a previous open.
  const [entered, setEntered] = useState(false)
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (!open) setEntered(false)
  }

  useEffect(() => {
    if (!open) return
    const frame = requestAnimationFrame(() => setEntered(true))
    return () => cancelAnimationFrame(frame)
  }, [open])

  if (!open) return null

  function handlePanelClick(event: MouseEvent<HTMLDivElement>) {
    event.stopPropagation()
  }

  return createPortal(
    <div className="fixed inset-0 z-50 bg-black/50" onClick={onClose}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={subtitle ? subtitleId : undefined}
        tabIndex={-1}
        onClick={handlePanelClick}
        onKeyDown={handleKeyDown}
        className={`fixed inset-y-0 right-0 flex h-full w-full flex-col bg-surface shadow-xl transition-transform duration-300 ease-out focus:outline-none ${MAX_WIDTH_CLASSES[maxWidth]} ${entered ? 'translate-x-0' : 'translate-x-full'}`}
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
              <div id={subtitleId} className="mt-1 text-sm text-text-muted">
                {subtitle}
              </div>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {headerActions}
            {onClose && (
              <IconButton icon={XMarkIcon} label="Close" variant="ghost" onClick={onClose} />
            )}
          </div>
        </div>

        <div className="min-w-0 flex-1 overflow-y-auto p-4 sm:p-6">{children}</div>
      </div>
    </div>,
    document.body
  )
}
