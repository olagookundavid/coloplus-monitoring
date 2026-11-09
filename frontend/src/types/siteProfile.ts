// Mirrors dto.SiteRequest/SiteDTO (backend/internal/dto/sites.go) -- the
// full site master record. Distinct from types/adminSite.ts's `AdminSite`
// (a 7-field read-only lookup shape used across ~8 pages) and
// types/site.ts's customer-facing, mock-backed `Site` -- this is the real,
// writable record behind POST/PUT /sites.

// Mirrors domain.SiteStatus.
export type SiteStatus = 'pending_acceptance' | 'active' | 'suspended'

export const SITE_STATUSES: SiteStatus[] = ['pending_acceptance', 'active', 'suspended']

export const SITE_STATUS_LABELS: Record<SiteStatus, string> = {
  pending_acceptance: 'Pending acceptance',
  active: 'Active',
  suspended: 'Suspended',
}

// Mirrors the power-source enum domain.Site.Validate() checks against.
export type PowerSource = 'grid' | 'generator' | 'battery' | 'solar' | 'building_supply'

export const POWER_SOURCES: PowerSource[] = [
  'grid',
  'generator',
  'battery',
  'solar',
  'building_supply',
]

export const POWER_SOURCE_LABELS: Record<PowerSource, string> = {
  grid: 'Grid',
  generator: 'Generator',
  battery: 'Battery',
  solar: 'Solar',
  building_supply: 'Building supply',
}

// Mirrors domain.UnreportedTimePolicy (DECISIONS 051).
export type UnreportedTimePolicy = 'available' | 'unknown'

export interface SiteProfile {
  id: string
  customerSiteCode: string
  coloplusCode: string
  name: string
  address: string
  region: string
  state: string
  siteType: string
  powerSources: PowerSource[]
  responsibleTeam: string
  unreportedTime: UnreportedTimePolicy
  /** Generator litres per running hour; absent until agreed. */
  consumptionLPH?: number
  tankCapacityLitres?: number
  /** Minutes from local midnight. */
  serviceStartMinute: number
  serviceEndMinute: number
  freshnessHours: number
  status: SiteStatus
  /** yyyy-mm-dd. */
  commissioningDate?: string
  latitude?: number
  longitude?: number
  generatorCapacity: string
  accessContactName: string
  accessContactPhone: string
  supervisorName: string
  createdAt: string
  updatedAt: string
}

// The writable shape -- identical for create and update, since PUT is a
// full replace (no PATCH exists).
export type SiteProfileInput = Omit<SiteProfile, 'id' | 'createdAt' | 'updatedAt'>
