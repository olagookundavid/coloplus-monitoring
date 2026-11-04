import type { CalculationRun } from '../types'

export function calculationRunLabel(run: CalculationRun): string {
  return run.label || `${run.periodFrom} – ${run.periodTo}`
}

// Never a fabricated 0 for a null percent field (frontend/CLAUDE.md §11) --
// null means the figure wasn't available for that site/period, not zero.
export function formatRunPercent(value: number | null): string {
  return value === null ? '—' : `${value.toFixed(1)}%`
}
