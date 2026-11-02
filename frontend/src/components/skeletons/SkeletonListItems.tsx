import { Skeleton } from './Skeleton'

export interface SkeletonListItemsProps {
  count?: number
  /** Set false for a plainer 2-line card (e.g. admin Overview's side lists),
      true for the badge + age row the master-detail pages' list cards have. */
  withBadge?: boolean
}

// Matches the hand-rolled <li> list-item cards used by the master-detail
// pages (Approvals, DailyReports, ReporterIncidents, Corrections) and, with
// withBadge={false}, admin Overview's two side list-cards. Returns a
// fragment of <li>s -- drops straight into the existing <ul>.
export function SkeletonListItems({ count = 4, withBadge = true }: SkeletonListItemsProps) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <li key={i} className="rounded-lg border border-border bg-surface p-4">
          {withBadge && (
            <div className="mb-2 flex items-center justify-between">
              <Skeleton rounded="full" height={20} className="w-16" />
              <Skeleton height={12} className="w-10" />
            </div>
          )}
          <Skeleton height={14} className="w-3/5" />
          <Skeleton height={11} className="mt-1.5 w-2/5" />
        </li>
      ))}
    </>
  )
}
