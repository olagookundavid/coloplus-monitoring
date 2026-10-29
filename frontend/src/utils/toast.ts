/**
 * Custom toast store (docs/DECISIONS.md 025 — a deliberate departure from
 * the Fusewall default, react-hot-toast). Callable from anywhere, not just
 * inside a component (e.g. a future axios interceptor), via a tiny
 * subscribable module-level store that ToastViewport reads with
 * useSyncExternalStore. Call-site shape matches what react-hot-toast would
 * have offered: toast.success(message), toast.error(message), etc.
 */

export type ToastTone = 'success' | 'error' | 'info' | 'warning'

export interface ToastEntry {
  id: string
  tone: ToastTone
  message: string
  duration: number
}

interface ToastOptions {
  duration?: number
}

const DEFAULT_DURATION: Record<ToastTone, number> = {
  success: 4000,
  info: 4000,
  warning: 6000,
  error: 6000,
}

let toasts: ToastEntry[] = []
const listeners = new Set<() => void>()

function emit(): void {
  for (const listener of listeners) listener()
}

function push(tone: ToastTone, message: string, options?: ToastOptions): string {
  const id = `toast_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const duration = options?.duration ?? DEFAULT_DURATION[tone]
  toasts = [...toasts, { id, tone, message, duration }]
  emit()
  if (duration > 0) {
    setTimeout(() => dismiss(id), duration)
  }
  return id
}

function dismiss(id: string): void {
  const next = toasts.filter((entry) => entry.id !== id)
  if (next.length === toasts.length) return
  toasts = next
  emit()
}

export const toast = {
  success: (message: string, options?: ToastOptions) => push('success', message, options),
  error: (message: string, options?: ToastOptions) => push('error', message, options),
  info: (message: string, options?: ToastOptions) => push('info', message, options),
  warning: (message: string, options?: ToastOptions) => push('warning', message, options),
  dismiss,
}

export function subscribeToToasts(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getToastSnapshot(): ToastEntry[] {
  return toasts
}
