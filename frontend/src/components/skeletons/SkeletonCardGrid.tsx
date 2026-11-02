import { Skeleton } from './Skeleton'

export interface SkeletonCardGridProps {
  count?: number
}

// Matches MySiteCard.tsx's exact DOM: title+badge row, subtitle, 7-segment
// day-bar row, 2-col mini-stat block, 2 full-width buttons. Used only by
// MySites.tsx, the one page whose main content is a card grid, not a table.
export function SkeletonCardGrid({ count = 6 }: SkeletonCardGridProps) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex flex-col gap-4 rounded-lg border border-border bg-surface p-5">
          <div className="flex items-start justify-between gap-2">
            <Skeleton height={16} className="w-24" />
            <Skeleton rounded="full" height={20} className="w-16" />
          </div>
          <Skeleton height={12} className="w-3/5" />
          <div className="flex gap-1">
            {Array.from({ length: 7 }).map((_, j) => (
              <Skeleton key={j} rounded="full" height={8} className="flex-1" />
            ))}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Skeleton height={20} className="w-12" />
            <Skeleton height={20} className="w-12" />
          </div>
          <div className="flex gap-2">
            <Skeleton height={40} className="flex-1" rounded="lg" />
            <Skeleton height={40} className="flex-1" rounded="lg" />
          </div>
        </div>
      ))}
    </div>
  )
}
