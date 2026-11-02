import { useEffect, useRef, useState } from 'react'

// Shared open/close + click-outside + Escape behavior for a trigger-button
// dropdown (AccountMenu, NotificationsMenu) -- not a dialog (no focus trap,
// no scroll lock; dialogBehavior.ts's useDialogLifecycle is for Modal/
// SidePanel, a different class of overlay).
export function useDropdownMenu<T extends HTMLElement = HTMLDivElement>() {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<T>(null)

  useEffect(() => {
    if (!open) return

    function handlePointerDown(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  return { open, setOpen, containerRef }
}
