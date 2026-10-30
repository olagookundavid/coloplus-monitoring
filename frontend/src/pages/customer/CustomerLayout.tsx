/* eslint-disable react-refresh/only-export-components -- CustomerLayout and
   its useDashboardFilters() hook are intentionally colocated, same pattern
   as AuthContext.tsx. */
import { useState } from 'react'
import { Outlet, useOutletContext } from 'react-router-dom'

import {
  CUSTOMER_NAV,
  CreateTicketModal,
  DashboardShell,
  PERIOD_PRESETS,
  REGION_OPTIONS,
} from '../../components/dashboard'

export interface CustomerLayoutContextValue {
  region: string
  setRegion: (region: string) => void
  period: string
  setPeriod: (period: string) => void
  openCreateTicket: (siteCode?: string) => void
}

// Filter state AND "is the Create ticket modal open" both persist/apply
// across customer-portal navigation because CustomerLayout is a React
// Router layout route (stays mounted across its child pages), not each
// page wrapping itself. Handed down via <Outlet context/> +
// useOutletContext() rather than a new contexts/ file — both are scoped to
// this one layout route, read only by its own children (Overview, Sites,
// SupportTickets), never truly global (architecture standard §4 reserves
// contexts/ for Auth-like global state).
export function CustomerLayout() {
  const [region, setRegion] = useState<string>(REGION_OPTIONS[0] ?? 'All regions')
  const [period, setPeriod] = useState<string>(PERIOD_PRESETS[0] ?? '')
  const [isCreateTicketOpen, setIsCreateTicketOpen] = useState(false)
  const [prefillSiteCode, setPrefillSiteCode] = useState<string | undefined>(undefined)

  const context: CustomerLayoutContextValue = {
    region,
    setRegion,
    period,
    setPeriod,
    openCreateTicket: (siteCode) => {
      setPrefillSiteCode(siteCode)
      setIsCreateTicketOpen(true)
    },
  }

  return (
    <DashboardShell nav={CUSTOMER_NAV}>
      <Outlet context={context} />
      <CreateTicketModal
        open={isCreateTicketOpen}
        onClose={() => setIsCreateTicketOpen(false)}
        initialSiteCode={prefillSiteCode}
      />
    </DashboardShell>
  )
}

export function useDashboardFilters(): CustomerLayoutContextValue {
  return useOutletContext<CustomerLayoutContextValue>()
}
