import { Skeleton } from './Skeleton'

// Matches the master-detail pages' (Approvals, DailyReports,
// ReporterIncidents, Corrections) right-pane general shape: badge+title,
// two fact blocks, an action row. One fixed shape works for all four --
// deliberately taller than the min-h-48 spinner box it replaces, to avoid a
// layout jump once the real, taller detail card loads.
export function SkeletonDetailPane() {
  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div>
        <Skeleton rounded="full" height={20} className="w-20" />
        <Skeleton height={20} className="mt-3 w-3/5" />
        <Skeleton height={12} className="mt-2 w-1/4" />
      </div>
      <div className="flex flex-col gap-3 rounded-lg border border-border p-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center justify-between">
            <Skeleton height={12} className="w-1/3" />
            <Skeleton height={12} className="w-1/5" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-3 rounded-lg border border-border p-4">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="flex items-center justify-between">
            <Skeleton height={12} className="w-1/3" />
            <Skeleton height={12} className="w-1/5" />
          </div>
        ))}
      </div>
      <div className="flex gap-3 border-t border-border pt-5">
        <Skeleton height={40} className="w-28" rounded="lg" />
        <Skeleton height={40} className="w-28" rounded="lg" />
      </div>
    </div>
  )
}
