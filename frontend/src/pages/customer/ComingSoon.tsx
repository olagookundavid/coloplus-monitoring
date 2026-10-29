import { ClockIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { ButtonLink } from '../../components/ui'
import { ROUTES } from '../../constants'

export interface ComingSoonProps {
  title: string
}

// Shared placeholder for Sites/Support tickets/Reports nav entries — a real,
// working link, not a dead or disabled nav item (frontend/CLAUDE.md §6: a
// nav item that looks clickable but silently does nothing is exactly the
// "secret button" trap that rule warns against).
export function ComingSoon({ title }: ComingSoonProps) {
  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Coloplus Monitoring"
        title={title}
        subtitle="This page isn't built yet."
      />
      <div className="flex flex-col items-start gap-3 rounded-lg border border-border bg-surface p-8">
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-soft text-primary-600">
          <ClockIcon className="h-5 w-5" aria-hidden="true" />
        </span>
        <p className="text-sm text-text-muted">
          {title} isn&rsquo;t built yet. Check back soon, or head back to the portfolio overview.
        </p>
        <ButtonLink to={ROUTES.HOME}>Back to Overview</ButtonLink>
      </div>
    </div>
  )
}
