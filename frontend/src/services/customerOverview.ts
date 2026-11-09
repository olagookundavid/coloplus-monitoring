import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  DataQuality,
  FuelBrief,
  PeriodSummary,
  PortfolioOverview,
  PortfolioQueryParams,
  PortfolioSite,
  PortfolioTrend,
  PortfolioTrendQueryParams,
  SiteCondition,
  SLA,
} from '../types'

interface DataQualityDTO {
  level: 'good' | 'fair' | 'poor'
  days: number
  reported_days: number
  completeness_pct: number
  late_reports: number
  corrections: number
  unconfirmed_share_pct: number
}

interface SLADTO {
  target_percent: number
  status: 'met' | 'at_risk' | 'missed'
  confirmed_meets_target: boolean
}

interface SummaryDTO {
  eligible_minutes: number
  served_minutes: number
  outage_minutes: number
  unknown_minutes: number
  coverage_pct: number | null
  reported_availability_pct: number | null
  assumed_available_minutes: number
  excluded_maintenance_minutes: number
  assumed_outage_minutes: number
  source_minutes?: Record<string, number>
  confirmed_served_share_pct: number | null
  provisional: boolean
}

interface FuelBriefDTO {
  basis: 'estimate'
  litres: number | null
  hours_remaining: number | null
  low: boolean
  empty: boolean
  provisional: boolean
}

interface SiteRowDTO {
  site_id: string
  customer_site_code: string
  coloplus_code: string
  name: string
  site_type: string
  state: string
  status: string
  condition: SiteCondition
  as_of?: string
  region: string
  responsible_team: string
  current_source?: string
  summary?: SummaryDTO
  days_reported: number
  days_without_report: number
  data_quality?: DataQualityDTO
  sla?: SLADTO
  fuel?: FuelBriefDTO
}

interface OverviewResponseDTO {
  date: string
  from: string
  to: string
  days: number
  region: string
  regions: string[]
  timezone: string
  generated_at: string
  active_sites: number
  pending_acceptance_sites: number
  suspended_sites: number
  sites_reported: number
  sites_not_reported: number
  sites_awaiting_update: number
  sites_outage_reported: number
  site_days_without_report: number
  data_quality?: DataQualityDTO
  sla?: SLADTO
  outages_this_week: number
  portfolio: SummaryDTO
  sites: SiteRowDTO[]
}

function mapDataQuality(dto: DataQualityDTO): DataQuality {
  return {
    level: dto.level,
    days: dto.days,
    reportedDays: dto.reported_days,
    completenessPct: dto.completeness_pct,
    lateReports: dto.late_reports,
    corrections: dto.corrections,
    unconfirmedSharePct: dto.unconfirmed_share_pct,
  }
}

function mapSLA(dto: SLADTO): SLA {
  return {
    targetPercent: dto.target_percent,
    status: dto.status,
    confirmedMeetsTarget: dto.confirmed_meets_target,
  }
}

function mapSummary(dto: SummaryDTO): PeriodSummary {
  return {
    eligibleMinutes: dto.eligible_minutes,
    servedMinutes: dto.served_minutes,
    outageMinutes: dto.outage_minutes,
    unknownMinutes: dto.unknown_minutes,
    coveragePct: dto.coverage_pct,
    reportedAvailabilityPct: dto.reported_availability_pct,
    assumedAvailableMinutes: dto.assumed_available_minutes,
    excludedMaintenanceMinutes: dto.excluded_maintenance_minutes,
    assumedOutageMinutes: dto.assumed_outage_minutes,
    sourceMinutes: dto.source_minutes,
    confirmedServedSharePct: dto.confirmed_served_share_pct,
    provisional: dto.provisional,
  }
}

function mapFuel(dto: FuelBriefDTO): FuelBrief {
  return {
    basis: dto.basis,
    litres: dto.litres,
    hoursRemaining: dto.hours_remaining,
    low: dto.low,
    empty: dto.empty,
    provisional: dto.provisional,
  }
}

function mapSite(dto: SiteRowDTO): PortfolioSite {
  return {
    id: dto.site_id,
    customerSiteCode: dto.customer_site_code,
    coloplusCode: dto.coloplus_code,
    name: dto.name,
    siteType: dto.site_type,
    state: dto.state,
    region: dto.region,
    status: dto.status,
    condition: dto.condition,
    asOf: dto.as_of,
    responsibleTeam: dto.responsible_team,
    currentSource: dto.current_source,
    summary: dto.summary ? mapSummary(dto.summary) : undefined,
    daysReported: dto.days_reported,
    daysWithoutReport: dto.days_without_report,
    dataQuality: dto.data_quality ? mapDataQuality(dto.data_quality) : undefined,
    sla: dto.sla ? mapSLA(dto.sla) : undefined,
    fuel: dto.fuel ? mapFuel(dto.fuel) : undefined,
  }
}

interface TrendPointDTO {
  date: string
  from: string
  to: string
  days: number
  active_sites: number
  sites_reported: number
  site_days: number
  site_days_reported: number
  summary: SummaryDTO
}

interface TrendResponseDTO {
  grain: 'day' | 'week' | 'month'
  from: string
  to: string
  region: string
  timezone: string
  formula: string
  version: string
  generated_at: string
  points: TrendPointDTO[]
}

export async function fetchPortfolioTrend(
  params: PortfolioTrendQueryParams
): Promise<PortfolioTrend> {
  try {
    const { data } = await getApi().get<TrendResponseDTO>('/dashboard/trend', { params })
    return {
      grain: data.grain,
      from: data.from,
      to: data.to,
      region: data.region,
      timezone: data.timezone,
      generatedAt: data.generated_at,
      points: data.points.map((p) => ({
        date: p.date,
        from: p.from,
        to: p.to,
        days: p.days,
        activeSites: p.active_sites,
        sitesReported: p.sites_reported,
        siteDays: p.site_days,
        siteDaysReported: p.site_days_reported,
        summary: mapSummary(p.summary),
      })),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the performance trend.')
  }
}

export async function fetchPortfolioOverview(
  params: PortfolioQueryParams = {}
): Promise<PortfolioOverview> {
  try {
    const { data } = await getApi().get<OverviewResponseDTO>('/dashboard/overview', { params })
    return {
      date: data.date,
      from: data.from,
      to: data.to,
      days: data.days,
      region: data.region,
      regions: data.regions,
      timezone: data.timezone,
      generatedAt: data.generated_at,
      activeSites: data.active_sites,
      pendingSites: data.pending_acceptance_sites,
      suspendedSites: data.suspended_sites,
      sitesReported: data.sites_reported,
      sitesNotReported: data.sites_not_reported,
      sitesAwaitingUpdate: data.sites_awaiting_update,
      sitesOutageReported: data.sites_outage_reported,
      siteDaysWithoutReport: data.site_days_without_report,
      dataQuality: data.data_quality ? mapDataQuality(data.data_quality) : undefined,
      sla: data.sla ? mapSLA(data.sla) : undefined,
      outagesThisWeek: data.outages_this_week,
      portfolio: mapSummary(data.portfolio),
      sites: data.sites.map(mapSite),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the portfolio overview.')
  }
}
