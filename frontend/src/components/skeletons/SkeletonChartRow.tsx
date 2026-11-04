import { Skeleton } from './Skeleton'

// A one-off matching customer Overview.tsx's chart card (header + Tabs +
// PerformanceTrendChart) specifically.
export function SkeletonChartRow() {
  return (
    <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <Skeleton height={16} className="w-40" />
          <Skeleton height={12} className="mt-2 w-56" />
        </div>
        <Skeleton height={32} className="w-32" rounded="full" />
      </div>
      <Skeleton height={224} className="mt-4 w-full" />
    </div>
  )
}
