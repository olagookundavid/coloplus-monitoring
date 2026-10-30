import type { Incident } from '../types'

/**
 * Mock-backed, same pattern as services/tickets.ts. Each incident tells the
 * same story as an existing ticket (Ikeja Central's diesel shortfall, Port
 * Harcourt's outage) rather than two unrelated fabrications for the same
 * site.
 */

const MOCK_INCIDENTS: Incident[] = [
  {
    id: 'FLT-1027',
    siteCode: 'LG0307',
    title: 'Diesel below reorder threshold',
    description: 'Fuel replenishment requested. Current estimate: 190 litres remaining.',
    impactNote: 'Service remains available. Refill required.',
    status: 'Open',
    occurredAt: '2026-10-24T08:40:00+01:00',
    owner: 'Coloplus Logistics',
    linkedTicketId: 'CP-2047',
  },
  {
    id: 'FLT-1014',
    siteCode: 'RV0256',
    title: 'Power supply interruption',
    description: 'Grid supply lost; generator failed to start automatically.',
    impactNote: 'Site running on battery reserve. Field technician dispatched.',
    status: 'Open',
    occurredAt: '2026-10-24T09:12:00+01:00',
    owner: 'Coloplus Field Operations',
    linkedTicketId: 'CP-2048',
  },
]

function mockDelay(minMs = 400, maxMs = 800): Promise<void> {
  const duration = minMs + Math.random() * (maxMs - minMs)
  return new Promise((resolve) => setTimeout(resolve, duration))
}

export async function getIncidents(): Promise<Incident[]> {
  await mockDelay()
  return MOCK_INCIDENTS
}
