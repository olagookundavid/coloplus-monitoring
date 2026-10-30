import { useDataSnapshot } from '../../hooks'

export interface DataRecencyIndicatorProps {
  label?: string
  className?: string
}

// Dot + text together, always (frontend/CLAUDE.md §11) — never a color
// alone standing in for "this data is current."
export function DataRecencyIndicator({
  label = 'As of',
  className = '',
}: DataRecencyIndicatorProps) {
  const snapshot = useDataSnapshot()
  return (
    <p className={`inline-flex items-center gap-1.5 text-sm text-text-muted ${className}`}>
      <span className="h-2 w-2 shrink-0 rounded-full bg-primary" aria-hidden="true" />
      {label} · {snapshot.timeLabel} WAT
    </p>
  )
}
