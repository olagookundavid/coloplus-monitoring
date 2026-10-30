import type { Site, Ticket } from '../types'
import { toCsv } from './csv'

export interface ReportScope {
  region: string
  period: string
}

function formatWatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Africa/Lagos',
  })
}

function buildPreamble(reportName: string, scope: ReportScope): Array<Array<string>> {
  return [
    ['Report', reportName],
    ['Generated', new Date().toISOString()],
    ['Scope', `${scope.region} · ${scope.period}`],
    ['Time standard', 'Africa/Lagos · WAT (UTC+1)'],
    [],
  ]
}

export function buildSitePerformanceCsv(sites: Site[], scope: ReportScope): string {
  const header = [
    'Site ID',
    'Site name',
    'Region',
    'Status',
    'Power source',
    'Energy (kWh)',
    'Diesel level (%)',
    'Power availability (%)',
    'Radio availability (%)',
    'Updated (WAT)',
  ]
  const rows = sites.map((site) => [
    site.code,
    site.name,
    site.region,
    site.status,
    site.powerSource,
    site.energyKwh,
    site.dieselPercent,
    site.powerAvailabilityPercent.toFixed(2),
    // null (not reported) stays a blank cell — never 0, never a repeated
    // value (frontend/CLAUDE.md §11, now enforced in the file too).
    site.radioAvailabilityPercent == null ? null : site.radioAvailabilityPercent.toFixed(2),
    formatWatTime(site.updatedAt),
  ])
  return toCsv([...buildPreamble('Site performance', scope), header, ...rows])
}

export function buildDieselFuelCsv(sites: Site[], scope: ReportScope): string {
  const header = ['Site ID', 'Site name', 'Power source', 'Diesel level (%)', 'Updated (WAT)']
  const rows = sites.map((site) => [
    site.code,
    site.name,
    site.powerSource,
    site.dieselPercent,
    formatWatTime(site.updatedAt),
  ])
  return toCsv([...buildPreamble('Diesel & fuel', scope), header, ...rows])
}

export function buildFaultsSupportCsv(tickets: Ticket[], scope: ReportScope): string {
  const header = [
    'Ticket ID',
    'Subject',
    'Site ID',
    'Category',
    'Priority',
    'Status',
    'Created (WAT)',
  ]
  const rows = tickets.map((ticket) => [
    ticket.id,
    ticket.subject,
    ticket.siteCode,
    ticket.category,
    ticket.priority,
    ticket.status,
    formatWatTime(ticket.createdAt),
  ])
  return toCsv([...buildPreamble('Faults & support', scope), header, ...rows])
}
