import type { DashboardOverview, ReportStatus, ReportStatusSummary } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface ReportStatusDTO {
  report_id: string
  status: ReportStatus
  row_version: number
  review_comment?: string
}

interface SiteRowDTO {
  site_id: string
  airtel_code: string
  coloplus_code: string
  name: string
  site_type: string
  state: string
  today_status?: ReportStatusDTO
  yesterday_status?: ReportStatusDTO
}

interface OverviewResponseDTO {
  date: string
  timezone: string
  generated_at: string
  outages_this_week: number
  sites: SiteRowDTO[]
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

// GET /dashboard/overview, the reporter-relevant fields only (today/
// yesterday status per site and the trailing-week outage count) -- the
// admin portfolio's approved-only availability figures aren't used here.
export async function fetchDashboardOverview(date?: string): Promise<DashboardOverview> {
  try {
    const { data } = await getApi().get<OverviewResponseDTO>('/dashboard/overview', {
      params: { date },
    })
    return {
      date: data.date,
      timezone: data.timezone,
      generatedAt: data.generated_at,
      outagesThisWeek: data.outages_this_week,
      sites: data.sites.map((site) => ({
        siteId: site.site_id,
        airtelCode: site.airtel_code,
        coloplusCode: site.coloplus_code,
        name: site.name,
        siteType: site.site_type,
        state: site.state,
        todayStatus: mapReportStatus(site.today_status),
        yesterdayStatus: mapReportStatus(site.yesterday_status),
      })),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load today’s overview.')
  }
}
