import { mapAcceptance } from './acceptance'
import { getApi } from './api'
import { mapAxiosError } from './auth'
import { mapDispute } from './disputes'
import { mapPublishedIncident } from './incidents'
import { mapIncident } from './reporterIncidents'
import type {
  Acceptance,
  Dispute,
  PublishedIncident,
  ReporterIncident,
  SiteCondition,
} from '../types'

// The whole site screen in one call (GET /sites/:id/overview). Sections the
// caller may not see are left out server-side -- which incident/dispute/
// acceptance shape comes back (and whether it's the staff or Airtel view)
// depends entirely on the caller's permissions, not on anything this file
// decides, so both the ops and the customer portal call the same
// fetchSiteOverview and just read the field that applies to them (incidents
// vs publishedIncidents).

export interface SiteProfile {
  id: string
  airtelCode: string
  coloplusCode: string
  name: string
  address: string
  region: string
  state: string
  siteType: string
  powerSources: string[]
  status: string
  commissioningDate?: string
  generatorCapacity: string
}

export interface SiteOverview {
  site?: SiteProfile
  timezone?: string
  condition: SiteCondition
  asOf?: string
  openIncidents: number
  /** Present for a caller with incidents.read (staff). */
  incidents: ReporterIncident[]
  /** Present for a caller without incidents.read (Airtel). */
  publishedIncidents: PublishedIncident[]
  openDisputes: number
  disputes: Dispute[]
  acceptance?: Acceptance
}

interface SiteDTO {
  id: string
  airtel_code: string
  coloplus_code: string
  name: string
  address: string
  region: string
  state: string
  site_type: string
  power_sources: string[]
  status: string
  commissioning_date?: string
  generator_capacity: string
}

interface SiteOverviewResponseDTO {
  site?: SiteDTO
  timezone?: string
  condition: SiteCondition
  as_of?: string
  open_incidents?: number
  incidents?: Parameters<typeof mapIncident>[0][]
  published_incidents?: Parameters<typeof mapPublishedIncident>[0][]
  open_disputes?: number
  disputes?: Parameters<typeof mapDispute>[0][]
  acceptance?: Parameters<typeof mapAcceptance>[0]
}

function mapSiteProfile(dto: SiteDTO): SiteProfile {
  return {
    id: dto.id,
    airtelCode: dto.airtel_code,
    coloplusCode: dto.coloplus_code,
    name: dto.name,
    address: dto.address,
    region: dto.region,
    state: dto.state,
    siteType: dto.site_type,
    powerSources: dto.power_sources,
    status: dto.status,
    commissioningDate: dto.commissioning_date,
    generatorCapacity: dto.generator_capacity,
  }
}

export async function fetchSiteOverview(siteId: string, days = 1): Promise<SiteOverview> {
  try {
    const { data } = await getApi().get<SiteOverviewResponseDTO>(`/sites/${siteId}/overview`, {
      params: { days },
    })
    return {
      site: data.site ? mapSiteProfile(data.site) : undefined,
      timezone: data.timezone,
      condition: data.condition,
      asOf: data.as_of,
      openIncidents: data.open_incidents ?? 0,
      incidents: (data.incidents ?? []).map(mapIncident),
      publishedIncidents: (data.published_incidents ?? []).map(mapPublishedIncident),
      openDisputes: data.open_disputes ?? 0,
      disputes: (data.disputes ?? []).map(mapDispute),
      acceptance: data.acceptance ? mapAcceptance(data.acceptance) : undefined,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this site.')
  }
}
