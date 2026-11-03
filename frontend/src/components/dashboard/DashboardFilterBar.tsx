/* eslint-disable react-refresh/only-export-components -- PERIOD_PRESETS and
   periodToDateRange are small static data/helpers colocated with the one
   component that uses them, same intentional pattern as AuthContext.tsx. */
import { CalendarIcon, MapPinIcon } from '@heroicons/react/24/outline'

import type { PortfolioQueryParams } from '../../types'
import { Select } from '../ui'
import { DataRecencyIndicator } from './DataRecencyIndicator'

export const ALL_REGIONS = 'All regions'

// Preset labels only -- the actual region list comes from the real
// GET /dashboard/overview response's `regions[]` (built server-side from
// the caller's scope), not a hardcoded guess at what they're called.
export const PERIOD_PRESETS = ['Today', 'Yesterday', 'Last 7 days', 'Last 30 days', 'This month']

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10)
}

// Translates a preset label into the real date|from&to params GET
// /dashboard/overview takes (Decision 043) -- the one place that
// knowledge lives, so every page reading these filters agrees.
export function periodToDateRange(period: string): Pick<PortfolioQueryParams, 'date' | 'from' | 'to'> {
  const today = new Date()
  switch (period) {
    case 'Yesterday': {
      const d = new Date(today)
      d.setDate(d.getDate() - 1)
      return { date: isoDate(d) }
    }
    case 'Last 7 days': {
      const from = new Date(today)
      from.setDate(from.getDate() - 6)
      return { from: isoDate(from), to: isoDate(today) }
    }
    case 'Last 30 days': {
      const from = new Date(today)
      from.setDate(from.getDate() - 29)
      return { from: isoDate(from), to: isoDate(today) }
    }
    case 'This month': {
      const from = new Date(today.getFullYear(), today.getMonth(), 1)
      return { from: isoDate(from), to: isoDate(today) }
    }
    default:
      return { date: isoDate(today) }
  }
}

export interface DashboardFilterBarProps {
  region: string
  onRegionChange: (region: string) => void
  regionOptions: string[]
  period: string
  onPeriodChange: (period: string) => void
}

export function DashboardFilterBar({
  region,
  onRegionChange,
  regionOptions,
  period,
  onPeriodChange,
}: DashboardFilterBarProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-col gap-2 sm:flex-row">
        <div className="w-full sm:w-44">
          <Select
            icon={MapPinIcon}
            value={region}
            onChange={(event) => onRegionChange(event.target.value)}
            aria-label="Region"
          >
            <option value="">{ALL_REGIONS}</option>
            {regionOptions.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </Select>
        </div>
        <div className="w-full sm:w-52">
          <Select
            icon={CalendarIcon}
            value={period}
            onChange={(event) => onPeriodChange(event.target.value)}
            aria-label="Period"
          >
            {PERIOD_PRESETS.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </Select>
        </div>
      </div>
      <DataRecencyIndicator />
    </div>
  )
}
