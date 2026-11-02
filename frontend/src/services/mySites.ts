import type { DayBarColor, MySitesResponse, ReportStatus, ReportStatusSummary } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface ReportStatusDTO {
  report_id: string
  status: ReportStatus
  row_version: number
  review_comment?: string
}

interface DayBarDTO {
  date: string
  color: DayBarColor
}

interface MySiteRowDTO {
  site_id: string
  airtel_code: string
  coloplus_code: string
  name: string
  site_type: string
  address: string
  state: string
  generator_capacity: string
  access_contact_name: string
  access_contact_phone: string
  supervisor_name: string
  today_status?: ReportStatusDTO
  availability_pct_30d: number | null
  open_incidents: number
  bars: DayBarDTO[]
}

interface MySitesResponseDTO {
  date: string
  timezone: string
  generated_at: string
  sites: MySiteRowDTO[]
}

function mapReportStatus(dto?: ReportStatusDTO): ReportStatusSummary | undefined {
  if (!dto) return undefined
  return {
    reportId: dto.report_id,
    status: dto.status,
    rowVersion: dto.row_version,
    reviewComment: dto.review_comment,
  }
}

// GET /dashboard/my-sites: the reporter's own site grid in one call --
// today's status, a trailing 7-day health strip, pooled 30-day availability
// and open-incident count per site, scoped to the signed-in reporter.
export async function fetchMySites(): Promise<MySitesResponse> {
  try {
    const { data } = await getApi().get<MySitesResponseDTO>('/dashboard/my-sites')
    return {
      date: data.date,
      timezone: data.timezone,
      generatedAt: data.generated_at,
      sites: data.sites.map((site) => ({
        siteId: site.site_id,
        airtelCode: site.airtel_code,
        coloplusCode: site.coloplus_code,
        name: site.name,
        siteType: site.site_type,
        address: site.address,
        state: site.state,
        generatorCapacity: site.generator_capacity,
        accessContactName: site.access_contact_name,
        accessContactPhone: site.access_contact_phone,
        supervisorName: site.supervisor_name,
        todayStatus: mapReportStatus(site.today_status),
        availabilityPct30d: site.availability_pct_30d,
        openIncidents: site.open_incidents,
        bars: site.bars.map((bar) => ({ date: bar.date, color: bar.color })),
      })),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load your sites.')
  }
}
