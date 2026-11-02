import { getApi } from './api'
import { mapAxiosError } from './auth'
import { mapIncident } from './reporterIncidents'
import type { ReporterIncident } from '../types'

// Trimmed to what the supervisor Sites detail panel shows today (live
// condition + open incidents). GET /sites/:id/overview also returns days,
// disputes and acceptance -- left unmapped since those pages are deferred
// (no frontend exists for disputes/acceptance anywhere yet); add fields
// here if a later round builds on them.

interface IncidentDTO {
  id: string
  ref: string
  site_id: string
  category: ReporterIncident['category']
  severity: ReporterIncident['severity']
  impact: ReporterIncident['impact']
  title: string
  description: string
  status: ReporterIncident['status']
  started_at: string
  detected_at: string
  owner_id?: string
  next_action?: string
  next_update_due?: string
  overdue: boolean
  restored_at?: string
  verified_at?: string
  closed_at?: string
  root_cause?: string
  published_at?: string
  created_at: string
  updated_at: string
}

interface SiteOverviewResponseDTO {
  condition: 'available' | 'outage_reported' | 'awaiting_update'
  as_of?: string
  open_incidents?: number
  incidents?: IncidentDTO[]
}

export interface SiteOverview {
  condition: 'available' | 'outage_reported' | 'awaiting_update'
  asOf?: string
  openIncidents: number
  incidents: ReporterIncident[]
}

export async function fetchSiteOverview(siteId: string): Promise<SiteOverview> {
  try {
    const { data } = await getApi().get<SiteOverviewResponseDTO>(`/sites/${siteId}/overview`, {
      params: { days: 1 },
    })
    return {
      condition: data.condition,
      asOf: data.as_of,
      openIncidents: data.open_incidents ?? 0,
      incidents: (data.incidents ?? []).map(mapIncident),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this site.')
  }
}
