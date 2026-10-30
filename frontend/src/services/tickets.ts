import type { CreateTicketRequest, Ticket } from '../types'

/**
 * Mock-backed, same pattern as services/auth.ts and services/sites.ts. One
 * addition: tickets created through the modal persist to localStorage — the
 * modal's own copy promises "Demo tickets are stored in this browser. They
 * are not sent to Coloplus," so this is a real requirement, not decoration.
 */

const TICKETS_STORAGE_KEY = 'coloplus.tickets.demo'

const BASE_TICKETS: Ticket[] = [
  {
    id: 'CP-2048',
    subject: 'Power supply interruption at Port Harcourt',
    siteCode: 'RV0256',
    category: 'Power outage',
    priority: 'Critical',
    status: 'In progress',
    contactEmail: 'ops@airtel.example',
    description:
      'Site reported a power supply interruption; generator failed to start automatically.',
    attachmentNames: [],
    createdAt: '2026-10-24T09:12:00+01:00',
  },
  {
    id: 'CP-2047',
    subject: 'Diesel replenishment request',
    siteCode: 'LG0307',
    category: 'Low fuel / diesel',
    priority: 'High',
    status: 'Open',
    contactEmail: 'ops@airtel.example',
    description: 'Tank level below threshold, requesting a scheduled refill.',
    attachmentNames: [],
    createdAt: '2026-10-24T08:48:00+01:00',
  },
]

function mockDelay(minMs = 400, maxMs = 800): Promise<void> {
  const duration = minMs + Math.random() * (maxMs - minMs)
  return new Promise((resolve) => setTimeout(resolve, duration))
}

function readStoredTickets(): Ticket[] {
  try {
    const raw = localStorage.getItem(TICKETS_STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as Ticket[]
  } catch {
    return []
  }
}

function writeStoredTickets(tickets: Ticket[]): void {
  try {
    localStorage.setItem(TICKETS_STORAGE_KEY, JSON.stringify(tickets))
  } catch {
    // Best-effort demo persistence (private browsing, storage quota) —
    // never blocks ticket creation if it throws.
  }
}

function nextTicketId(existing: Ticket[]): string {
  const maxNumber = existing.reduce((max, ticket) => {
    const match = /^CP-(\d+)$/.exec(ticket.id)
    const num = match?.[1] ? Number(match[1]) : 0
    return Math.max(max, num)
  }, 0)
  return `CP-${maxNumber + 1}`
}

export async function getTickets(): Promise<Ticket[]> {
  await mockDelay()
  return [...BASE_TICKETS, ...readStoredTickets()]
}

export async function createTicket(request: CreateTicketRequest): Promise<Ticket> {
  await mockDelay()
  const stored = readStoredTickets()
  const ticket: Ticket = {
    id: nextTicketId([...BASE_TICKETS, ...stored]),
    subject: request.subject,
    siteCode: request.siteCode,
    category: request.category,
    priority: request.priority,
    status: 'Open',
    contactEmail: request.contactEmail,
    description: request.description,
    attachmentNames: request.attachmentNames,
    createdAt: new Date().toISOString(),
  }
  writeStoredTickets([...stored, ticket])
  return ticket
}
