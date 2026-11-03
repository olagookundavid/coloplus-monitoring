import { useState } from 'react'
import { ChevronDownIcon, InformationCircleIcon } from '@heroicons/react/24/outline'

// frontend/CLAUDE.md §12: every derived metric exposes a "How calculated"
// affordance (formula id/version, inputs, warnings). This is that shared
// primitive -- first used by the fuel estimate (FuelPanel), generic for
// later reuse by report/formula calculations. No "approver" row: unlike a
// report calculation, a live fuel estimate has no approval workflow -- omit
// the field rather than passing a value, don't assume it's mandatory here.
export interface CalculationDetailRow {
  label: string
  value: string
}

export interface CalculationDetailsProps {
  formula: string
  formulaVersion: string
  explanation: string
  rows: CalculationDetailRow[]
  warnings?: string[]
  className?: string
}

export function CalculationDetails({
  formula,
  formulaVersion,
  explanation,
  rows,
  warnings = [],
  className = '',
}: CalculationDetailsProps) {
  const [open, setOpen] = useState(false)

  return (
    <div className={className}>
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-expanded={open}
        className="inline-flex items-center gap-1.5 text-sm font-medium text-primary-700 hover:underline"
      >
        <InformationCircleIcon className="h-4 w-4" aria-hidden="true" />
        How calculated
        <ChevronDownIcon
          className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden="true"
        />
      </button>

      {open && (
        <div className="mt-3 flex flex-col gap-3 rounded-lg border border-border bg-surface-subtle p-4 text-sm">
          <p className="text-text-muted">{explanation}</p>
          <dl className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between gap-3">
              <dt className="text-text-muted">Formula</dt>
              <dd className="font-medium text-text">
                {formula} · v{formulaVersion}
              </dd>
            </div>
            {rows.map((row) => (
              <div key={row.label} className="flex items-center justify-between gap-3">
                <dt className="text-text-muted">{row.label}</dt>
                <dd className="font-medium text-text">{row.value}</dd>
              </div>
            ))}
          </dl>
          {warnings.length > 0 && (
            <ul className="flex flex-col gap-1 border-t border-border pt-3">
              {warnings.map((warning) => (
                <li key={warning} className="text-warning-700">
                  {warning}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
