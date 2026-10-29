/* eslint-disable react-refresh/only-export-components -- CustomerLayout and
   its useDashboardFilters() hook are intentionally colocated, same pattern
   as AuthContext.tsx. */
import { useState } from 'react'
import { Outlet, useOutletContext } from 'react-router-dom'

import {
  CUSTOMER_NAV,
  DashboardShell,
  PERIOD_PRESETS,
  REGION_OPTIONS,
} from '../../components/dashboard'

export interface DashboardFiltersContextValue {
  region: string
  setRegion: (region: string) => void
  period: string
  setPeriod: (period: string) => void
}

// Filter state persists across customer-portal navigation because
// CustomerLayout is a React Router layout route (stays mounted across its
// child pages), not each page wrapping itself. Handed down via
// <Outlet context/> + useOutletContext() rather than a new contexts/ file —
// this is one layout route's scoped state, not truly global (architecture
// standard §4 reserves contexts/ for Auth-like global state).
export function CustomerLayout() {
  const [region, setRegion] = useState<string>(REGION_OPTIONS[0] ?? 'All regions')
  const [period, setPeriod] = useState<string>(PERIOD_PRESETS[0] ?? '')

  const context: DashboardFiltersContextValue = { region, setRegion, period, setPeriod }

  return (
    <DashboardShell nav={CUSTOMER_NAV}>
      <Outlet context={context} />
    </DashboardShell>
  )
}

export function useDashboardFilters(): DashboardFiltersContextValue {
  return useOutletContext<DashboardFiltersContextValue>()
}
