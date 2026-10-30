import type { ComponentType, ReactNode, SVGProps } from 'react'

export interface StatCardTrend {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  label: string
  tone: 'positive' | 'caution'
}

export interface StatCardProps {
  label: string
  value: ReactNode
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** Omit when there's no real trend to show (a status count, not a
      measurement over time) — use `caption` alone instead of inventing one. */
  trend?: StatCardTrend
  /** What the number's basis/period is — PRD §5/§7's "every metric exposes
      its definition," not decoration. Optional: some cards (Support tickets)
      only need the single trend line, not a second muted caption line. */
  caption?: ReactNode
}

const TREND_TONE_CLASS: Record<StatCardTrend['tone'], string> = {
  positive: 'text-primary-700',
  caution: 'text-warning-700',
}

export function StatCard({ label, value, icon: Icon, trend, caption }: StatCardProps) {
  const TrendIcon = trend?.icon
  return (
    <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-medium text-text-muted">{label}</p>
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
      </div>
      <p className="mt-2 text-2xl font-bold text-text">{value}</p>
      {trend && TrendIcon && (
        <p
          className={`mt-1 inline-flex items-center gap-1 text-sm font-medium ${TREND_TONE_CLASS[trend.tone]}`}
        >
          <TrendIcon className="h-4 w-4" aria-hidden="true" />
          {trend.label}
        </p>
      )}
      {caption && <p className="mt-1 text-xs text-text-muted">{caption}</p>}
    </div>
  )
}
