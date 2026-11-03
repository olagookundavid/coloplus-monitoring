import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import type { PortfolioTrendPoint } from '../../types'

export interface PerformanceTrendChartProps {
  points: PortfolioTrendPoint[]
}

// One-off chart accents (frontend/CLAUDE.md §7's documented-exception
// pattern). CONFIRMED_COLOR is the real primary-600 token's own hex, not a
// copy. REPORTED_COLOR has no semantic token to reuse (warning/danger would
// wrongly imply something's wrong with a perfectly normal series) -- this
// exact pair was run through the dataviz skill's validate_palette.js
// (light mode, this app has no dark theme): lightness band, chroma floor,
// CVD separation (ΔE 13.8 protan) and normal-vision floor (ΔE 28.8) all
// pass, plus the line style (solid/dashed) is a second, color-independent
// encoding of the same distinction.
const CONFIRMED_COLOR = 'var(--color-primary-600)'
const REPORTED_COLOR = '#ea580c'

interface TrendChartRow {
  date: string
  reportedAvailabilityPct: number | null
  confirmedServedSharePct: number | null
}

// Dates arrive as plain YYYY-MM-DD with no time/zone component (unlike a
// timestamp). A bare "YYYY-MM-DD" string parses as UTC midnight, which can
// render as the previous day in a timezone behind UTC -- appending a
// zone-less time parses it as local midnight instead, so the day never shifts.
function formatDateTick(date: string): string {
  return new Date(`${date}T00:00:00`).toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
  })
}

function formatPct(value: number | null): string {
  return value == null ? '—' : `${value.toFixed(2)}%`
}

interface TrendTooltipPayloadEntry {
  dataKey: string
  name?: string
  value: number | null
  color?: string
}

interface TrendTooltipProps {
  active?: boolean
  label?: string
  payload?: TrendTooltipPayloadEntry[]
  points: PortfolioTrendPoint[]
}

function TrendTooltip({ active, label, payload, points }: TrendTooltipProps) {
  if (!active || !payload || payload.length === 0 || !label) return null
  const point = points.find((p) => p.date === label)
  return (
    <div className="rounded-lg border border-border bg-surface p-3 text-sm shadow-lg">
      <p className="mb-1.5 font-medium text-text">{formatDateTick(label)}</p>
      <div className="flex flex-col gap-1">
        {payload.map((entry) => (
          <p key={entry.dataKey} className="flex items-center gap-2">
            <span
              className="h-0.5 w-3 shrink-0 rounded"
              style={{ backgroundColor: entry.color }}
              aria-hidden="true"
            />
            <span className="font-semibold text-text">{formatPct(entry.value)}</span>
            <span className="text-text-muted">{entry.name}</span>
          </p>
        ))}
      </div>
      {point && (
        <p className="mt-1.5 border-t border-border pt-1.5 text-xs text-text-muted">
          {point.sitesReported} of {point.activeSites} sites reported
        </p>
      )}
    </div>
  )
}

export function PerformanceTrendChart({ points }: PerformanceTrendChartProps) {
  const data: TrendChartRow[] = points.map((p) => ({
    date: p.date,
    reportedAvailabilityPct: p.summary.reportedAvailabilityPct,
    confirmedServedSharePct: p.summary.confirmedServedSharePct,
  }))

  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-x-6 gap-y-1.5 text-sm">
        <span className="inline-flex items-center gap-1.5">
          <span
            className="inline-block h-0.5 w-3 align-middle"
            style={{
              backgroundImage: `repeating-linear-gradient(to right, ${REPORTED_COLOR} 0 4px, transparent 4px 7px)`,
            }}
            aria-hidden="true"
          />
          <span className="font-medium text-text">Reported availability</span>
          <span className="text-text-muted">
            — includes minutes nobody reported, counted as available by default
          </span>
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span
            className="h-0.5 w-3 rounded"
            style={{ backgroundColor: CONFIRMED_COLOR }}
            aria-hidden="true"
          />
          <span className="font-medium text-text">Confirmed served share</span>
          <span className="text-text-muted">— confirmed only, never includes assumed time</span>
        </span>
      </div>

      <ResponsiveContainer width="100%" height={240}>
        <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <CartesianGrid horizontal vertical={false} stroke="var(--color-border)" strokeWidth={1} />
          <XAxis
            dataKey="date"
            tickFormatter={formatDateTick}
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 12, fill: 'var(--color-text-muted)' }}
          />
          <YAxis
            domain={[0, 100]}
            tickFormatter={(value: number) => `${value}%`}
            axisLine={false}
            tickLine={false}
            width={40}
            tick={{ fontSize: 12, fill: 'var(--color-text-muted)' }}
          />
          <Tooltip content={<TrendTooltip points={points} />} />
          <Line
            dataKey="reportedAvailabilityPct"
            name="Reported availability"
            stroke={REPORTED_COLOR}
            strokeWidth={2}
            strokeDasharray="6 4"
            dot={{ r: 3 }}
            connectNulls={false}
            isAnimationActive={false}
          />
          <Line
            dataKey="confirmedServedSharePct"
            name="Confirmed served share"
            stroke={CONFIRMED_COLOR}
            strokeWidth={2}
            dot={{ r: 3 }}
            connectNulls={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
