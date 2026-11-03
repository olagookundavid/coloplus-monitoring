import { Button, StatusBadge } from '../ui'
import type { DayBar, MySiteRow } from '../../types'
import { actionFor, statusLabel } from '../../utils'

export interface MySiteCardProps {
  site: MySiteRow
  onReport: () => void
  onDetails: () => void
  /** Rendered only when passed -- a page without fuel permissions omits it. */
  onFuel?: () => void
}

const BAR_CLASS: Record<DayBar['color'], string> = {
  green: 'bg-primary',
  amber: 'bg-warning',
  red: 'bg-danger',
}

const BAR_DESCRIPTION: Record<DayBar['color'], string> = {
  green: 'reported, no outage',
  amber: 'reported, had an outage',
  red: 'no report filed',
}

function formatBarDate(iso: string): string {
  const [, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(2000, (m ?? 1) - 1, d ?? 1)).toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
    timeZone: 'UTC',
  })
}

function formatAvailability(pct: number | null): string {
  return pct == null ? '—' : `${Math.round(pct)}%`
}

export function MySiteCard({ site, onReport, onDetails, onFuel }: MySiteCardProps) {
  const { label, tone } = statusLabel(site.todayStatus?.status)
  const { label: actionLabel, mode } = actionFor(site)
  const reportedDays = site.bars.filter((bar) => bar.color !== 'red').length

  return (
    <div className="flex min-w-0 flex-col gap-4 rounded-lg border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-base font-semibold text-text">{site.airtelCode}</h3>
        <StatusBadge label={label} tone={tone} />
      </div>
      <p className="-mt-2 truncate text-sm text-text-muted">
        {site.address || 'Address not available yet'} · {site.siteType}
      </p>

      <div>
        <p className="text-xs font-medium uppercase tracking-wide text-text-muted">Last 7 days</p>
        <div className="mt-2 flex gap-1">
          {site.bars.map((bar) => (
            <span
              key={bar.date}
              role="img"
              aria-label={`${formatBarDate(bar.date)}: ${BAR_DESCRIPTION[bar.color]}`}
              title={`${formatBarDate(bar.date)}: ${BAR_DESCRIPTION[bar.color]}`}
              className={`h-2 flex-1 rounded-full ${BAR_CLASS[bar.color]}`}
            />
          ))}
        </div>
        <p className="mt-1 text-xs text-text-muted">
          {reportedDays} of {site.bars.length} days reported
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <p className="text-xs text-text-muted">Power availability, 30 days</p>
          <p className="mt-1 text-lg font-bold text-text">
            {formatAvailability(site.availabilityPct30d)}
          </p>
        </div>
        <div>
          <p className="text-xs text-text-muted">Open incidents</p>
          <p className="mt-1 text-lg font-bold text-text">{site.openIncidents}</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <div className="min-w-28 flex-1">
          <Button variant={mode === 'view' ? 'outline' : 'primary'} fullWidth onClick={onReport}>
            {mode === 'create'
              ? 'Report today'
              : mode === 'edit'
                ? `${actionLabel} report`
                : 'View report'}
          </Button>
        </div>
        <div className="min-w-28 flex-1">
          <Button variant="outline" fullWidth onClick={onDetails}>
            Site details
          </Button>
        </div>
        {onFuel && (
          <div className="min-w-28 flex-1">
            <Button variant="outline" fullWidth onClick={onFuel}>
              Fuel
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
