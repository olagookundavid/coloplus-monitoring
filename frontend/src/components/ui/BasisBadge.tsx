import { StatusBadge, type StatusTone } from './StatusBadge'

// frontend/CLAUDE.md §12: every derived metric carries a Basis badge, via a
// shared primitive, never a one-off colored span. This is that primitive --
// first used by the fuel estimate (FuelPanel), generic for later reuse by
// report/formula calculations.
export type Basis = 'measured' | 'calculated' | 'estimated' | 'unavailable'

export interface BasisBadgeProps {
  basis: Basis
  className?: string
}

const BASIS_LABEL: Record<Basis, string> = {
  measured: 'Measured',
  calculated: 'Calculated',
  estimated: 'Estimated',
  unavailable: 'Unavailable',
}

const BASIS_TONE: Record<Basis, StatusTone> = {
  measured: 'success',
  calculated: 'info',
  estimated: 'warning',
  unavailable: 'neutral',
}

export function BasisBadge({ basis, className }: BasisBadgeProps) {
  return <StatusBadge label={BASIS_LABEL[basis]} tone={BASIS_TONE[basis]} className={className} />
}
