import {
  Area,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import type { PerformanceTrendPoint, TrendMetric } from '../../types'

export interface PerformanceTrendChartProps {
  data: PerformanceTrendPoint[]
  metric: TrendMetric
}

// One-off chart accents, no brand token for either (frontend/CLAUDE.md §7
// known-gap pattern, same as Button's success/info fallbacks). Colors
// validated with the dataviz skill's contrast/CVD checker, not eyeballed:
// orange-600 (not the more obvious orange-500) is the one that actually
// clears legible contrast and colorblind separation against the teal.
const POWER_COLOR = '#0d9488'
const RADIO_COLOR = '#ea580c'

function formatHourTick(timestamp: string): string {
  return new Date(timestamp).toLocaleTimeString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Africa/Lagos',
  })
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
  metric: TrendMetric
}

function TrendTooltip({ active, label, payload, metric }: TrendTooltipProps) {
  if (!active || !payload || payload.length === 0 || !label) return null
  return (
    <div className="rounded-lg border border-border bg-surface p-3 text-sm shadow-lg">
      <p className="mb-1.5 font-medium text-text">{formatHourTick(label)}</p>
      <div className="flex flex-col gap-1">
        {payload.map((entry) => (
          <p key={entry.dataKey} className="flex items-center gap-2">
            <span className="h-0.5 w-3 shrink-0 rounded" style={{ backgroundColor: entry.color }} />
            <span className="font-semibold text-text">
              {entry.value == null
                ? '—'
                : metric === 'availability'
                  ? `${entry.value}%`
                  : `${entry.value} kWh`}
            </span>
            <span className="text-text-muted">{entry.name}</span>
          </p>
        ))}
      </div>
    </div>
  )
}

export function PerformanceTrendChart({ data, metric }: PerformanceTrendChartProps) {
  return (
    <div>
      {/* Legend / basis caption — text+dot, never color-matching alone
          (frontend/CLAUDE.md §11). Energy's caption also carries the
          reported-vs-estimated distinction visible on the line itself. */}
      {metric === 'availability' ? (
        <div className="mb-2 flex flex-wrap gap-4 text-sm text-text-muted">
          <span className="inline-flex items-center gap-1.5">
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: POWER_COLOR }}
              aria-hidden="true"
            />
            Power availability
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: RADIO_COLOR }}
              aria-hidden="true"
            />
            Radio network
          </span>
        </div>
      ) : (
        <p className="mb-2 text-sm text-text-muted">
          <span
            className="mr-1.5 inline-block h-0.5 w-3 align-middle"
            style={{
              backgroundImage: `repeating-linear-gradient(to right, ${POWER_COLOR} 0 4px, transparent 4px 7px)`,
            }}
            aria-hidden="true"
          />
          Estimated · approved load profile
        </p>
      )}

      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <CartesianGrid horizontal vertical={false} stroke="var(--color-border)" strokeWidth={1} />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatHourTick}
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 12, fill: 'var(--color-text-muted)' }}
          />
          <YAxis
            domain={metric === 'availability' ? [90, 100] : ['auto', 'auto']}
            tickFormatter={(value: number) =>
              metric === 'availability' ? `${value}%` : `${value}`
            }
            axisLine={false}
            tickLine={false}
            width={48}
            tick={{ fontSize: 12, fill: 'var(--color-text-muted)' }}
          />
          <Tooltip content={<TrendTooltip metric={metric} />} />
          {metric === 'availability' ? (
            <>
              <defs>
                <linearGradient id="powerAvailabilityFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={POWER_COLOR} stopOpacity={0.12} />
                  <stop offset="100%" stopColor={POWER_COLOR} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Area
                dataKey="powerAvailabilityPercent"
                stroke="none"
                fill="url(#powerAvailabilityFill)"
                connectNulls={false}
                isAnimationActive={false}
                legendType="none"
              />
              <Line
                dataKey="powerAvailabilityPercent"
                name="Power availability"
                stroke={POWER_COLOR}
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
              <Line
                dataKey="radioAvailabilityPercent"
                name="Radio network"
                stroke={RADIO_COLOR}
                strokeWidth={2}
                dot={false}
                connectNulls={false}
                isAnimationActive={false}
              />
            </>
          ) : (
            <Line
              dataKey="energyKwh"
              name="Energy"
              stroke={POWER_COLOR}
              strokeWidth={2}
              strokeDasharray="6 4"
              dot={false}
              connectNulls={false}
              isAnimationActive={false}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
