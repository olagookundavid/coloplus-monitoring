import { useEffect, type KeyboardEvent, type RefObject } from 'react'

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

export function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => el.offsetParent !== null
  )
}

/**
 * Shared behavior for any full-screen dialog (Modal, SidePanel): focus the
 * dialog on open and restore focus to whatever triggered it on close,
 * reentrant-safe body-scroll-lock (restores the *previous* overflow value,
 * not '', so a dialog stacking on top of another won't prematurely unlock
 * scroll when it closes first), and a keydown handler that closes on Escape
 * and wraps Tab inside the dialog.
 */
export function useDialogLifecycle(
  open: boolean,
  dialogRef: RefObject<HTMLElement | null>,
  onClose?: () => void
) {
  // Focus the dialog itself on open, not the first field — autofocusing a
  // <select> can pop its options open in some browsers, and it would skip
  // announcing the title first.
  useEffect(() => {
    if (!open) return
    const previouslyFocused = document.activeElement as HTMLElement | null
    dialogRef.current?.focus()
    return () => {
      previouslyFocused?.focus()
    }
  }, [open, dialogRef])

  useEffect(() => {
    if (!open) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = previousOverflow
    }
  }, [open])

  function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
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

  return { handleKeyDown }
}
