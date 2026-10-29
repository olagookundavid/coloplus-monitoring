import { useSyncExternalStore } from 'react'

import { getToastSnapshot, subscribeToToasts } from '../../utils'
import { Toast } from './Toast'

export function ToastViewport() {
  const toasts = useSyncExternalStore(subscribeToToasts, getToastSnapshot, getToastSnapshot)

  if (toasts.length === 0) return null

  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed inset-x-4 top-4 z-50 flex flex-col gap-2 sm:inset-x-auto sm:right-4 sm:w-96"
    >
      {toasts.map((entry) => (
        <div key={entry.id} className="pointer-events-auto">
          <Toast entry={entry} />
        </div>
      ))}
    </div>
  )
}
