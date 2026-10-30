import type { AdminSite } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

// The real GET /sites endpoint -- separate from services/sites.ts, which
// stays customer-only and mock-backed (DECISIONS 039), a different, much
// richer display shape than the real SiteDTO.
interface SiteDTO {
  id: string
  airtel_code: string
  coloplus_code: string
  name: string
  site_type: string
  region: string
  state: string
}

function mapAdminSite(dto: SiteDTO): AdminSite {
  return {
    id: dto.id,
    airtelCode: dto.airtel_code,
    coloplusCode: dto.coloplus_code,
    name: dto.name,
    siteType: dto.site_type,
    region: dto.region,
    state: dto.state,
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
