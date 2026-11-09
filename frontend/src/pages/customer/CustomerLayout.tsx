/* eslint-disable react-refresh/only-export-components -- CustomerLayout and
   its useDashboardFilters() hook are intentionally colocated, same pattern
   as AuthContext.tsx. */
import { useState } from 'react'
import { Outlet, useOutletContext } from 'react-router-dom'

import {
  CUSTOMER_NAV_GROUPS,
  CreateTicketModal,
  DashboardShell,
  PERIOD_PRESETS,
} from '../../components/dashboard'
import { usePermissions } from '../../hooks'

export interface CustomerLayoutContextValue {
  /** Empty string means every region -- the real GET /dashboard/overview's own meaning for "no filter." */
  region: string
  setRegion: (region: string) => void
  period: string
  setPeriod: (period: string) => void
  /** Empty string means every quality level -- GET /dashboard/overview's own meaning for "no filter." */
  quality: string
  setQuality: (quality: string) => void
  openCreateTicket: (siteId?: string) => void
}

// Filter state AND "is the Create ticket modal open" both persist/apply
// across customer-portal navigation because CustomerLayout is a React
// Router layout route (stays mounted across its child pages), not each
// page wrapping itself. Handed down via <Outlet context/> +
// useOutletContext() rather than a new contexts/ file — both are scoped to
// this one layout route, read only by its own children (Overview, Sites,
// SupportTickets), never truly global (architecture standard §4 reserves
// contexts/ for Auth-like global state).
// The customer's own co-brand mark — fixed here only, never reused as an app
// color elsewhere (frontend/CLAUDE.md §7). No real logo asset yet, same
// provisional caveat as the Coloplus wordmark.
const CUSTOMER_BRAND_MARK = <span className="text-lg font-bold italic text-red-600">Customer</span>

export function CustomerLayout() {
  const { can } = usePermissions()
  // disputes.raise is the one permission that separates reviewer
  // from viewer (Decision 034) -- every other customer-portal
  // permission is shared, so this single check is enough.
  const portalLabel = can('disputes.raise') ? 'Reviewer portal' : 'Viewer portal'

  const [region, setRegion] = useState<string>('')
  const [period, setPeriod] = useState<string>(PERIOD_PRESETS[0] ?? '')
  const [quality, setQuality] = useState<string>('')
  const [isCreateTicketOpen, setIsCreateTicketOpen] = useState(false)
  const [prefillSiteId, setPrefillSiteId] = useState<string | undefined>(undefined)

  const context: CustomerLayoutContextValue = {
    region,
    setRegion,
    period,
    setPeriod,
    quality,
    setQuality,
    openCreateTicket: (siteId) => {
      setPrefillSiteId(siteId)
      setIsCreateTicketOpen(true)
    },
  }

  return (
    <DashboardShell
      nav={CUSTOMER_NAV_GROUPS}
      portalLabel={portalLabel}
      brandMark={CUSTOMER_BRAND_MARK}
    >
      <Outlet context={context} />
      <CreateTicketModal
        open={isCreateTicketOpen}
        onClose={() => setIsCreateTicketOpen(false)}
        initialSiteId={prefillSiteId}
      />
    </DashboardShell>
  )
}

export function useDashboardFilters(): CustomerLayoutContextValue {
  return useOutletContext<CustomerLayoutContextValue>()
}
