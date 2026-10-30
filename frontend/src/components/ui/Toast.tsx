import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline'

import { toast, type ToastEntry } from '../../utils'
import { IconButton } from './IconButton'

const TONE_ICON = {
  success: CheckCircleIcon,
  error: XCircleIcon,
  warning: ExclamationTriangleIcon,
  // info falls back to a neutral icon/palette — no brand token sourced
  // yet for "info" (frontend/CLAUDE.md §7 known gap).
  info: InformationCircleIcon,
} as const

const TONE_CLASSES = {
  success: 'border-primary bg-primary-soft text-primary-700',
  error: 'border-danger bg-danger-50 text-danger-600',
  warning: 'border-warning bg-warning-soft text-warning-500',
  info: 'border-border bg-surface-subtle text-text',
} as const

export interface ToastProps {
  entry: ToastEntry
}

export function Toast({ entry }: ToastProps) {
  const Icon = TONE_ICON[entry.tone]

  return (
    <div
      role="status"
      className={`flex items-start gap-3 rounded-lg border bg-surface p-3 shadow-lg ${TONE_CLASSES[entry.tone]}`}
    >
      <Icon className="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
      <p className="flex-1 text-sm text-text">{entry.message}</p>
      <IconButton
        icon={XMarkIcon}
        label="Dismiss"
        variant="ghost"
        onClick={() => toast.dismiss(entry.id)}
        className="-my-2.5 -mr-2.5"
      />
    </div>
  )
}
