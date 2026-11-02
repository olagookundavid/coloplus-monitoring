import { Skeleton } from './Skeleton'

export interface SkeletonStatCardsProps {
  count: 3 | 4
}

const GRID_CLASS: Record<3 | 4, string> = {
  3: 'grid-cols-1 gap-4 sm:grid-cols-3',
  4: 'grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4',
}

// Matches StatCard.tsx's DOM: label + icon circle row, value line, caption
// line. Covers both stat-card grid patterns used anywhere in the app.
export function SkeletonStatCards({ count }: SkeletonStatCardsProps) {
  return (
    <div className={`grid ${GRID_CLASS[count]}`}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="min-w-0 rounded-lg border border-border bg-surface p-5">
          <div className="flex items-start justify-between gap-2">
            <Skeleton height={14} className="w-2/5" />
            <Skeleton rounded="full" width={36} height={36} />
          </div>
          <Skeleton height={28} className="mt-3 w-24" />
          <Skeleton height={12} className="mt-2 w-32" />
        </div>
      ))}
    </div>
  )
}
