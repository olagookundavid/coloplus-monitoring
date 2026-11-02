import { Skeleton } from './Skeleton'

// A one-off matching customer Overview.tsx's chart card (header + Tabs +
// PerformanceTrendChart) and FuelHealthCard row specifically.
export function SkeletonChartRow() {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div className="min-w-0 rounded-lg border border-border bg-surface p-5 lg:col-span-2">
        <div className="flex items-start justify-between gap-3">
          <div>
            <Skeleton height={16} className="w-40" />
            <Skeleton height={12} className="mt-2 w-56" />
          </div>
          <Skeleton height={32} className="w-32" rounded="full" />
        </div>
        <Skeleton height={224} className="mt-4 w-full" />
      </div>
      <div className="flex flex-col gap-4 rounded-lg border border-border bg-surface p-5">
        <div className="flex items-start justify-between gap-2">
          <div>
            <Skeleton height={16} className="w-24" />
            <Skeleton height={12} className="mt-2 w-32" />
          </div>
          <Skeleton rounded="full" width={36} height={36} />
        </div>
        <Skeleton height={28} className="w-28" />
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i}>
            <div className="flex items-center justify-between">
              <Skeleton height={12} className="w-20" />
              <Skeleton height={12} className="w-8" />
            </div>
            <Skeleton height={6} className="mt-1 w-full" rounded="full" />
          </div>
        ))}
      </div>
    </div>
  )
}
