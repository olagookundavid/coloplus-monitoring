export type MaintenanceStatus = 'requested' | 'approved' | 'rejected' | 'cancelled'

export interface MaintenanceRules {
  noticeHours: number
  maxHours: number
  monthlyCapHours: number
}

export interface MaintenanceWindow {
  id: string
  siteId: string
  siteName?: string
  startsAt: string
  endsAt: string
  minutes: number
  reason: string
  status: MaintenanceStatus
  /** Omitted for an viewer/reviewer -- who asked and who decided is
      Coloplus-internal. Always present for every ops-portal role. */
  requestedBy?: string
  requestedByName?: string
  requestedAt: string
  decidedBy?: string
  decidedByName?: string
  decidedAt?: string
  decisionNote?: string
}

export interface MaintenanceListResult {
  windows: MaintenanceWindow[]
  rules: MaintenanceRules
  total: number
  limit: number
  offset: number
}

export interface MaintenanceFilter {
  siteId?: string
  status?: MaintenanceStatus
  from?: string
  to?: string
  limit?: number
  offset?: number
}
