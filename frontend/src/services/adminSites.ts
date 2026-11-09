import type {
  AdminSite,
  PowerSource,
  SiteProfile,
  SiteProfileInput,
  SiteStatus,
  UnreportedTimePolicy,
} from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

// The real GET /sites endpoint -- separate from services/sites.ts, which
// stays customer-only and mock-backed (DECISIONS 039). SiteDTO here is the
// full wire shape (backend/internal/dto/sites.go); fetchAdminSites only
// reads a 7-field subset of it for the many pages that just need a
// lightweight lookup, while fetchSiteProfiles/create/update below use the
// rest for the site-management page.
interface SiteDTO {
  id: string
  customer_site_code: string
  coloplus_code: string
  name: string
  address: string
  region: string
  state: string
  site_type: string
  power_sources: PowerSource[]
  responsible_team: string
  unreported_time: UnreportedTimePolicy
  consumption_lph?: number
  tank_capacity_litres?: number
  service_start_minute: number
  service_end_minute: number
  freshness_hours: number
  status: SiteStatus
  commissioning_date?: string
  latitude?: number
  longitude?: number
  generator_capacity: string
  access_contact_name: string
  access_contact_phone: string
  supervisor_name: string
  created_at: string
  updated_at: string
}

function mapAdminSite(dto: SiteDTO): AdminSite {
  return {
    id: dto.id,
    customerSiteCode: dto.customer_site_code,
    coloplusCode: dto.coloplus_code,
    name: dto.name,
    siteType: dto.site_type,
    region: dto.region,
    state: dto.state,
  }
}

function mapSiteProfile(dto: SiteDTO): SiteProfile {
  return {
    id: dto.id,
    customerSiteCode: dto.customer_site_code,
    coloplusCode: dto.coloplus_code,
    name: dto.name,
    address: dto.address,
    region: dto.region,
    state: dto.state,
    siteType: dto.site_type,
    powerSources: dto.power_sources,
    responsibleTeam: dto.responsible_team,
    unreportedTime: dto.unreported_time,
    consumptionLPH: dto.consumption_lph,
    tankCapacityLitres: dto.tank_capacity_litres,
    serviceStartMinute: dto.service_start_minute,
    serviceEndMinute: dto.service_end_minute,
    freshnessHours: dto.freshness_hours,
    status: dto.status,
    commissioningDate: dto.commissioning_date,
    latitude: dto.latitude,
    longitude: dto.longitude,
    generatorCapacity: dto.generator_capacity,
    accessContactName: dto.access_contact_name,
    accessContactPhone: dto.access_contact_phone,
    supervisorName: dto.supervisor_name,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

function toSiteRequestBody(input: SiteProfileInput) {
  return {
    customer_site_code: input.customerSiteCode,
    coloplus_code: input.coloplusCode,
    name: input.name,
    address: input.address,
    region: input.region,
    state: input.state,
    site_type: input.siteType,
    power_sources: input.powerSources,
    responsible_team: input.responsibleTeam,
    unreported_time: input.unreportedTime,
    consumption_lph: input.consumptionLPH,
    tank_capacity_litres: input.tankCapacityLitres,
    service_start_minute: input.serviceStartMinute,
    service_end_minute: input.serviceEndMinute,
    freshness_hours: input.freshnessHours,
    status: input.status,
    commissioning_date: input.commissioningDate,
    latitude: input.latitude,
    longitude: input.longitude,
    generator_capacity: input.generatorCapacity,
    access_contact_name: input.accessContactName,
    access_contact_phone: input.accessContactPhone,
    supervisor_name: input.supervisorName,
  }
}

export async function fetchAdminSites(): Promise<AdminSite[]> {
  try {
    const { data } = await getApi().get<{ sites: SiteDTO[] }>('/sites', {
      params: { limit: 200 },
    })
    return data.sites.map(mapAdminSite)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load sites.')
  }
}

export async function fetchSiteProfiles(): Promise<SiteProfile[]> {
  try {
    const { data } = await getApi().get<{ sites: SiteDTO[] }>('/sites', {
      params: { limit: 200 },
    })
    return data.sites.map(mapSiteProfile)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load sites.')
  }
}

export async function createSiteProfile(input: SiteProfileInput): Promise<SiteProfile> {
  try {
    const { data } = await getApi().post<SiteDTO>('/sites', toSiteRequestBody(input))
    return mapSiteProfile(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not create this site.')
  }
}

export async function updateSiteProfile(id: string, input: SiteProfileInput): Promise<SiteProfile> {
  try {
    const { data } = await getApi().put<SiteDTO>(`/sites/${id}`, toSiteRequestBody(input))
    return mapSiteProfile(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not save this site.')
  }
}
