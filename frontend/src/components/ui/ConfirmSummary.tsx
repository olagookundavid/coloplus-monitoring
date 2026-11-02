import type { ReactNode } from 'react'

export interface ConfirmSummaryRow {
  label: string
  value: ReactNode
}

export interface ConfirmSummaryProps {
  title: ReactNode
  meta?: ReactNode
  rows?: ConfirmSummaryRow[]
}

// Names the specific record a confirmation acts on -- the site, period or
// record, never just "this item" (frontend/CLAUDE.md §2).
export function ConfirmSummary({ title, meta, rows }: ConfirmSummaryProps) {
  return (
    <div className="rounded-lg border border-border bg-surface-subtle p-3">
      <p className="font-medium text-text">{title}</p>
      {meta && <p className="mt-0.5 text-sm text-text-muted">{meta}</p>}
      {rows && rows.length > 0 && (
        <dl className="mt-2 flex flex-col gap-1">
          {rows.map((row) => (
            <div key={row.label} className="flex items-center justify-between gap-2 text-sm">
              <dt className="text-text-muted">{row.label}</dt>
              <dd className="font-medium text-text">{row.value}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  )
}
