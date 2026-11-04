import { useSyncExternalStore } from 'react'

import { getToastSnapshot, subscribeToToasts } from '../../utils'
import { Toast } from './Toast'

export function ToastViewport() {
  const toasts = useSyncExternalStore(subscribeToToasts, getToastSnapshot, getToastSnapshot)

  if (toasts.length === 0) return null

  return (
    <div
      aria-live="polite"
      // z-[60], one above every other fixed-position container in the app
      // (Modal/SidePanel/MainLoader all use z-50) -- a toast must stay
      // visible over an open modal or side panel, not get painted under it.
      className="pointer-events-none fixed inset-x-4 top-4 z-[60] flex flex-col gap-2 sm:inset-x-auto sm:right-4 sm:w-96"
    >
      {toasts.map((entry) => (
        <div key={entry.id} className="pointer-events-auto">
          <Toast entry={entry} />
        </div>
      ))}
    </div>
  )
}
