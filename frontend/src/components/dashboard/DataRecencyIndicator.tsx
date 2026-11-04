import { formatTimeWAT } from '../../utils'

export interface DataRecencyIndicatorProps {
  /** The real ISO timestamp this data was generated at. Omit (or pass
      undefined while a query is loading) to render nothing — never a
      fabricated or stale time (frontend/CLAUDE.md §11). */
  isoTimestamp?: string
  label?: string
  className?: string
}

// Dot + text together, always (frontend/CLAUDE.md §11) — never a color
// alone standing in for "this data is current."
export function DataRecencyIndicator({
  isoTimestamp,
  label = 'As of',
  className = '',
}: DataRecencyIndicatorProps) {
  if (!isoTimestamp) return null
  return (
    <p className={`inline-flex items-center gap-1.5 text-sm text-text-muted ${className}`}>
      <span className="h-2 w-2 shrink-0 rounded-full bg-primary" aria-hidden="true" />
      {label} · {formatTimeWAT(isoTimestamp)} WAT
    </p>
  )
}
