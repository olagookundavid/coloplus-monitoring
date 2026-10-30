export type StatusTone = 'success' | 'warning' | 'danger' | 'neutral' | 'info'

export interface StatusBadgeProps {
  label: string
  tone: StatusTone
  className?: string
}

// Dot + text, always together (frontend/CLAUDE.md §11) — never color alone.
// "info" has no sourced brand token yet (§7 known gap), falls back to plain
// sky, same as Button's info variant.
const TONE_CLASSES: Record<StatusTone, { dot: string; text: string }> = {
  success: { dot: 'bg-primary', text: 'text-primary-700' },
  warning: { dot: 'bg-warning', text: 'text-warning-700' },
  danger: { dot: 'bg-danger', text: 'text-danger-600' },
  neutral: { dot: 'bg-slate-400', text: 'text-text-muted' },
  info: { dot: 'bg-sky-500', text: 'text-sky-700' },
}

export function StatusBadge({ label, tone, className = '' }: StatusBadgeProps) {
  const toneClasses = TONE_CLASSES[tone]
  return (
    <span
      className={`inline-flex items-center gap-1.5 text-sm font-medium ${toneClasses.text} ${className}`}
    >
      <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${toneClasses.dot}`} aria-hidden="true" />
      {label}
    </span>
  )
}
