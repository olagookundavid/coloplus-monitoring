import { useEffect, useId, useRef, useState, type ComponentType, type SVGProps } from 'react'
import { createPortal } from 'react-dom'
import { EllipsisVerticalIcon } from '@heroicons/react/24/outline'

// frontend/CLAUDE.md §4: a table row with real mutating actions ends in an
// Actions column holding this -- a three-dot button opening a dropdown, no
// inline action buttons, an action that can't run stays in the menu
// disabled with a hint. Documented there but not yet built anywhere; this
// is the first consumer (the fuel entry history's Void action), kept
// generic for reuse by every other table that gets mutating row actions.
export interface RowAction {
  label: string
  onClick: () => void
  icon?: ComponentType<SVGProps<SVGSVGElement>>
  tone?: 'default' | 'danger'
  disabled?: boolean
  /** Shown when disabled, explaining why this action can't run right now. */
  hint?: string
}

export interface RowActionsProps {
  actions: RowAction[]
  label?: string
}

interface MenuPosition {
  top: number
  right: number
}

// Portaled to document.body with `fixed` coordinates computed from the
// trigger button, rather than a plain in-flow `absolute` descendant of the
// trigger -- a row's three-dot button sits inside DataTable's own
// horizontally-scrolling wrapper, and a menu that stayed nested there could
// get tangled up in that ancestor's own layout/overflow (the menu's box
// counting toward it, or getting clipped) depending on which row it opened
// from. Portaling it out -- the same technique Modal/SidePanel already use
// to escape their own containers -- removes it from the table's layout
// entirely, so opening it can never change the table's height.
export function RowActions({ actions, label = 'Row actions' }: RowActionsProps) {
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState<MenuPosition | null>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  function openMenu() {
    const rect = triggerRef.current?.getBoundingClientRect()
    if (!rect) return
    setPosition({ top: rect.bottom + 4, right: window.innerWidth - rect.right })
    setOpen(true)
  }

  useEffect(() => {
    if (!open) return

    function handlePointerDown(event: MouseEvent) {
      const target = event.target as Node
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) return
      setOpen(false)
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    // A scroll (the page, or the table's own horizontal scroller) can leave
    // a `fixed`-positioned menu pointing at empty space -- closing it is
    // simpler and more honest than tracking the trigger's position live.
    function handleReposition() {
      setOpen(false)
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    window.addEventListener('scroll', handleReposition, true)
    window.addEventListener('resize', handleReposition)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('scroll', handleReposition, true)
      window.removeEventListener('resize', handleReposition)
    }
  }, [open])

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => (open ? setOpen(false) : openMenu())}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={label}
        title={label}
        className="inline-flex h-11 w-11 min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
      >
        <EllipsisVerticalIcon className="h-5 w-5" aria-hidden="true" />
      </button>

      {open &&
        position &&
        createPortal(
          <div
            ref={menuRef}
            id={menuId}
            role="menu"
            style={{ top: position.top, right: position.right }}
            className="fixed z-20 w-56 overflow-hidden rounded-lg border border-border bg-surface p-1.5 shadow-lg"
          >
            {actions.map((action) => (
              <button
                key={action.label}
                type="button"
                role="menuitem"
                disabled={action.disabled}
                title={action.disabled ? action.hint : undefined}
                onClick={() => {
                  setOpen(false)
                  action.onClick()
                }}
                className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent ${
                  action.tone === 'danger'
                    ? 'text-danger-600 hover:bg-danger-50'
                    : 'text-text hover:bg-surface-subtle'
                }`}
              >
                {action.icon && <action.icon className="h-4 w-4 shrink-0" aria-hidden="true" />}
                <span className="flex-1">{action.label}</span>
              </button>
            ))}
          </div>,
          document.body
        )}
    </>
  )
}
