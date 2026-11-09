// Mirrors dto.OutageItemDTO/OutageListResponse (GET /dashboard/outages) --
// the outage stretches behind a dashboard availability figure, each merged
// so no minute is counted twice, with the report it came from.
export interface Outage {
  siteId: string
  customerSiteCode: string
  coloplusCode: string
  siteName: string
  region: string
  start: string
  end: string
  minutes: number
  reportId: string
  reportDate: string
  reportRevision: number
  /** Coloplus-only; absent for a customer caller. */
  reason?: string
}

export interface OutageListParams {
  from: string
  to: string
  region?: string
  siteId?: string
  limit?: number
  offset?: number
}

export interface OutageListResult {
  from: string
  to: string
  timezone: string
  generatedAt: string
  /** Covers every outage in the period, not just this page. */
  totalMinutes: number
  total: number
  limit: number
  offset: number
  outages: Outage[]
}

// Mirrors dto.IncidentOutageDTO/IncidentOutagesResponse (GET
// /incidents/:id/outages) -- the outage the approved daily reports show
// during one incident's window. Nothing here is added to the daily figures
// a second time.
export interface IncidentOutage {
  start: string
  end: string
  minutes: number
  reportId: string
  reportDate: string
  reportRevision: number
}

export interface IncidentOutages {
  incidentId: string
  from: string
  to: string
  /** The incident is not restored yet, so the window runs to now. */
  open: boolean
  totalMinutes: number
  outages: IncidentOutage[]
}
