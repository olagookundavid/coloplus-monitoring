import type { ComponentType, SVGProps } from 'react'
import { EllipsisVerticalIcon } from '@heroicons/react/24/outline'

import { useDropdownMenu } from '../../hooks'

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

export function RowActions({ actions, label = 'Row actions' }: RowActionsProps) {
  const { open, setOpen, containerRef } = useDropdownMenu<HTMLDivElement>()

  return (
    <div ref={containerRef} className="relative inline-flex">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={label}
        title={label}
        className="inline-flex h-11 w-11 min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
      >
        <EllipsisVerticalIcon className="h-5 w-5" aria-hidden="true" />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-1 w-56 overflow-hidden rounded-lg border border-border bg-surface p-1.5 shadow-lg"
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
        </div>
      )}
    </div>
  )
}
