import type { FuelHealthSummary, PerformanceTrendPoint, PortfolioSummary, Site } from '../types'

/**
 * Mock-backed, same pattern as services/auth.ts: real future call
 * signatures, canned data, a fake delay. The backend has no site/power
 * domain yet (re-confirmed this round) — see docs/DECISIONS.md.
 */

export interface PortfolioQueryParams {
  region?: string
  period?: string
}

export interface PortfolioOverview {
  summary: PortfolioSummary
  trend: PerformanceTrendPoint[]
  fuelHealth: FuelHealthSummary
}

function mockDelay(minMs = 400, maxMs = 800): Promise<void> {
  const duration = minMs + Math.random() * (maxMs - minMs)
  return new Promise((resolve) => setTimeout(resolve, duration))
}

const MOCK_SITES: Site[] = [
  {
    id: 'site_lg0421',
    code: 'LG0421',
    name: 'Victoria Island',
    region: 'Lagos',
    status: 'Available',
    powerSource: 'Grid',
    energyKwh: 342,
    dieselPercent: 76,
    powerAvailabilityPercent: 100.0,
    radioAvailabilityPercent: 99.98,
    updatedAt: '2026-10-24T10:42:00+01:00',
  },
  {
    id: 'site_lg0182',
    code: 'LG0182',
    name: 'Lekki Phase 1',
    region: 'Lagos',
    status: 'Available',
    powerSource: 'Generator',
    energyKwh: 298,
    dieselPercent: 68,
    powerAvailabilityPercent: 99.92,
    radioAvailabilityPercent: 99.84,
    updatedAt: '2026-10-24T10:38:00+01:00',
  },
  {
    id: 'site_lg0307',
    code: 'LG0307',
    name: 'Ikeja Central',
    region: 'Lagos',
    status: 'Low diesel',
    powerSource: 'Generator',
    energyKwh: 325,
    dieselPercent: 19,
    powerAvailabilityPercent: 99.71,
    radioAvailabilityPercent: 99.63,
    updatedAt: '2026-10-24T10:35:00+01:00',
  },
  {
    id: 'site_og0074',
    code: 'OG0074',
    name: 'Ota Industrial',
    region: 'Ogun',
    status: 'Available',
    powerSource: 'Battery',
    energyKwh: 264,
    dieselPercent: 47,
    powerAvailabilityPercent: 97.36,
    radioAvailabilityPercent: 98.51,
    updatedAt: '2026-10-24T10:31:00+01:00',
  },
  {
    id: 'site_ab0118',
    code: 'AB0118',
    name: 'Wuse District',
    region: 'Abuja',
    status: 'Available',
    powerSource: 'Grid',
    energyKwh: 386,
    dieselPercent: 84,
    powerAvailabilityPercent: 100.0,
    radioAvailabilityPercent: 100.0,
    updatedAt: '2026-10-24T10:41:00+01:00',
  },
  {
    id: 'site_rv0256',
    code: 'RV0256',
    name: 'Port Harcourt',
    region: 'Rivers',
    status: 'Power outage',
    powerSource: 'No supply',
    energyKwh: 277,
    dieselPercent: 54,
    powerAvailabilityPercent: 93.42,
    radioAvailabilityPercent: 96.19,
    updatedAt: '2026-10-24T10:40:00+01:00',
  },
  {
    id: 'site_oy0093',
    code: 'OY0093',
    name: 'Ibadan North',
    region: 'Oyo',
    status: 'Available',
    powerSource: 'Grid',
    energyKwh: 241,
    dieselPercent: 61,
    powerAvailabilityPercent: 99.84,
    radioAvailabilityPercent: 99.92,
    updatedAt: '2026-10-24T10:36:00+01:00',
  },
  {
    id: 'site_kn0167',
    code: 'KN0167',
    name: 'Kano Municipal',
    region: 'Kano',
    status: 'Update overdue',
    powerSource: 'Awaiting update',
    energyKwh: 308,
    dieselPercent: 58,
    powerAvailabilityPercent: 98.87,
    radioAvailabilityPercent: null,
    updatedAt: '2026-10-24T07:15:00+01:00',
  },
]

const MOCK_SUMMARY: PortfolioSummary = {
  energyConsumedKwh: 2441,
  dieselRemainingLitres: 5924,
  sitesBelowDieselThreshold: 1,
  powerAvailabilityPercent: 98.64,
  sitesReportedAvailableCount: 6,
  radioAvailabilityPercent: 99.15,
  sitesWithRadioDataCount: 7,
  totalSitesInView: 8,
}

const MOCK_TREND: PerformanceTrendPoint[] = [
  {
    timestamp: '2026-10-24T00:00:00+01:00',
    powerAvailabilityPercent: 99.8,
    radioAvailabilityPercent: 99.5,
    energyKwh: 0,
  },
  {
    timestamp: '2026-10-24T01:00:00+01:00',
    powerAvailabilityPercent: 99.9,
    radioAvailabilityPercent: 99.5,
    energyKwh: 210,
  },
  {
    timestamp: '2026-10-24T02:00:00+01:00',
    powerAvailabilityPercent: 99.9,
    radioAvailabilityPercent: 99.6,
    energyKwh: 430,
  },
  {
    timestamp: '2026-10-24T03:00:00+01:00',
    powerAvailabilityPercent: 99.8,
    radioAvailabilityPercent: 98.9,
    energyKwh: 625,
  },
  {
    timestamp: '2026-10-24T04:00:00+01:00',
    powerAvailabilityPercent: 99.7,
    radioAvailabilityPercent: 98.3,
    energyKwh: 810,
  },
  {
    timestamp: '2026-10-24T05:00:00+01:00',
    powerAvailabilityPercent: 99.8,
    radioAvailabilityPercent: 98.8,
    energyKwh: 1020,
  },
  {
    timestamp: '2026-10-24T06:00:00+01:00',
    powerAvailabilityPercent: 99.9,
    radioAvailabilityPercent: 99.4,
    energyKwh: 1255,
  },
  {
    timestamp: '2026-10-24T07:00:00+01:00',
    powerAvailabilityPercent: 98.9,
    radioAvailabilityPercent: 99.4,
    energyKwh: 1480,
  },
  {
    timestamp: '2026-10-24T08:00:00+01:00',
    powerAvailabilityPercent: 97.2,
    radioAvailabilityPercent: 99.3,
    energyKwh: 1690,
  },
  {
    timestamp: '2026-10-24T09:00:00+01:00',
    powerAvailabilityPercent: 98.0,
    radioAvailabilityPercent: 99.3,
    energyKwh: 2080,
  },
  {
    timestamp: '2026-10-24T10:00:00+01:00',
    powerAvailabilityPercent: 98.9,
    radioAvailabilityPercent: 99.3,
    energyKwh: 2441,
  },
]

const MOCK_FUEL_HEALTH: FuelHealthSummary = {
  totalLitresRemaining: 5924,
  sites: [
    { siteId: 'site_lg0307', siteName: 'Ikeja Central', percent: 19 },
    { siteId: 'site_og0074', siteName: 'Ota Industrial', percent: 47 },
    { siteId: 'site_rv0256', siteName: 'Port Harcourt', percent: 54 },
    { siteId: 'site_kn0167', siteName: 'Kano Municipal', percent: 58 },
  ],
  sitesNeedingReplenishment: 1,
}

export async function getPortfolioOverview(
  _params: PortfolioQueryParams = {}
): Promise<PortfolioOverview> {
  await mockDelay()
  return { summary: MOCK_SUMMARY, trend: MOCK_TREND, fuelHealth: MOCK_FUEL_HEALTH }
}

export async function getSitePerformance(_params: PortfolioQueryParams = {}): Promise<Site[]> {
  await mockDelay()
  return MOCK_SITES
}
