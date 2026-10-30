/* eslint-disable react-refresh/only-export-components -- REGION_OPTIONS and
   PERIOD_PRESETS are small static data colocated with the one component
   that uses them, same intentional pattern as AuthContext.tsx. */
import { CalendarIcon, MapPinIcon } from '@heroicons/react/24/outline'

import { Select } from '../ui'
import { DataRecencyIndicator } from './DataRecencyIndicator'

// Hardcoded static content (frontend/CLAUDE.md §8) — no date-picker
// dependency added; these are presets with nothing real to pick against
// yet (mock data only). Revisit once a real backend period boundary exists.
export const REGION_OPTIONS = ['All regions', 'Lagos', 'Ogun', 'Abuja', 'Rivers', 'Oyo', 'Kano']
export const PERIOD_PRESETS = [
  'Today · 24 Oct 2026',
  'Yesterday',
  'Last 7 days',
  'Last 30 days',
  'This month',
]

export interface DashboardFilterBarProps {
  region: string
  onRegionChange: (region: string) => void
  period: string
  onPeriodChange: (period: string) => void
}

export function DashboardFilterBar({
  region,
  onRegionChange,
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
            {REGION_OPTIONS.map((option) => (
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
