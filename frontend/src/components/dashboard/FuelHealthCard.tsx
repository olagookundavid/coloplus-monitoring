import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import type { FuelHealthSummary } from '../../types'
import { FuelIcon } from '../icons'
import { ButtonLink } from '../ui'

export interface FuelHealthCardProps {
  summary: FuelHealthSummary
}

export function FuelHealthCard({ summary }: FuelHealthCardProps) {
  return (
    <div className="flex h-full min-w-0 flex-col rounded-lg border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h2 className="text-base font-semibold text-text">Fuel health</h2>
          <p className="text-sm text-text-muted">Latest tank levels across your sites</p>
        </div>
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
          <FuelIcon className="h-5 w-5" aria-hidden="true" />
        </span>
      </div>

      <p className="mt-3 text-2xl font-bold text-text">
        {summary.totalLitresRemaining.toLocaleString()}{' '}
        <span className="text-base font-medium text-text-muted">litres remaining</span>
      </p>

      <ul className="mt-4 flex flex-1 flex-col gap-3">
        {summary.sites.map((site) => {
          const isLow = site.percent < 25
          return (
            <li key={site.siteId}>
              <div className="flex items-center justify-between text-sm">
                <span className="text-text">{site.siteName}</span>
                <span className={isLow ? 'font-medium text-warning-700' : 'text-text-muted'}>
                  {site.percent}%
                </span>
              </div>
              <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-surface-subtle">
                <div
                  className={`h-full rounded-full ${isLow ? 'bg-warning' : 'bg-primary'}`}
                  style={{ width: `${site.percent}%` }}
                />
              </div>
            </li>
          )
        })}
      </ul>

      {summary.sitesNeedingReplenishment > 0 && (
        <div className="mt-4 flex items-center justify-between gap-2 border-t border-border pt-3 text-sm">
          <span className="inline-flex items-center gap-1.5 text-warning-700">
            <ExclamationTriangleIcon className="h-4 w-4" aria-hidden="true" />
            Replenishment needed at {summary.sitesNeedingReplenishment} site
            {summary.sitesNeedingReplenishment === 1 ? '' : 's'}
          </span>
          <ButtonLink to={ROUTES.CUSTOMER_SITES} variant="link" size="sm">
            Review
          </ButtonLink>
        </div>
      )}
    </div>
  )
}
