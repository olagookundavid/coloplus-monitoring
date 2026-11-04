import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { IncidentOutages, Outage, OutageListParams, OutageListResult } from '../types'

interface OutageItemDTO {
  site_id: string
  airtel_code: string
  coloplus_code: string
  site_name: string
  region: string
  start: string
  end: string
  minutes: number
  report_id: string
  report_date: string
  report_revision: number
  reason?: string
}

interface OutageListResponseDTO {
  from: string
  to: string
  timezone: string
  generated_at: string
  total_minutes: number
  total: number
  limit: number
  offset: number
  outages: OutageItemDTO[]
}

function mapOutage(dto: OutageItemDTO): Outage {
  return {
    siteId: dto.site_id,
    airtelCode: dto.airtel_code,
    coloplusCode: dto.coloplus_code,
    siteName: dto.site_name,
    region: dto.region,
    start: dto.start,
    end: dto.end,
    minutes: dto.minutes,
    reportId: dto.report_id,
    reportDate: dto.report_date,
    reportRevision: dto.report_revision,
    reason: dto.reason,
  }
}

export async function fetchOutages(params: OutageListParams): Promise<OutageListResult> {
  try {
    const { data } = await getApi().get<OutageListResponseDTO>('/dashboard/outages', {
      params: {
        from: params.from,
        to: params.to,
        region: params.region,
        site_id: params.siteId,
        limit: params.limit,
        offset: params.offset,
      },
    })
    return {
      from: data.from,
      to: data.to,
      timezone: data.timezone,
      generatedAt: data.generated_at,
      totalMinutes: data.total_minutes,
      total: data.total,
      limit: data.limit,
      offset: data.offset,
      outages: data.outages.map(mapOutage),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the outage history.')
  }
}

interface IncidentOutageDTO {
  start: string
  end: string
  minutes: number
  report_id: string
  report_date: string
  report_revision: number
}

interface IncidentOutagesResponseDTO {
  incident_id: string
  from: string
  to: string
  open: boolean
  total_minutes: number
  outages: IncidentOutageDTO[]
}

export async function fetchIncidentOutages(incidentId: string): Promise<IncidentOutages> {
  try {
    const { data } = await getApi().get<IncidentOutagesResponseDTO>(
      `/incidents/${incidentId}/outages`
    )
    return {
      incidentId: data.incident_id,
      from: data.from,
      to: data.to,
      open: data.open,
      totalMinutes: data.total_minutes,
      outages: data.outages.map((o) => ({
        start: o.start,
        end: o.end,
        minutes: o.minutes,
        reportId: o.report_id,
        reportDate: o.report_date,
        reportRevision: o.report_revision,
      })),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the outage behind this incident.')
  }
}
