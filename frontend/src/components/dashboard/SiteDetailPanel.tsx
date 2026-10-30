import { useState } from 'react'
import {
  ArrowRightIcon,
  ArrowTrendingUpIcon,
  BoltIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  MapPinIcon,
  PlusIcon,
  SignalIcon,
} from '@heroicons/react/24/outline'
import { Link } from 'react-router-dom'

import { FuelIcon } from '../icons'
import { Button, SidePanel, StatusBadge, Tabs, type StatusTone, type TabItem } from '../ui'
import { ticketDetailPath } from '../../constants'
import { useIncidents, useTickets } from '../../hooks'
import type { Incident, IncidentStatus, Site } from '../../types'
import { formatDateTimeWAT, getSiteStatusTone, getTicketStatusTone } from '../../utils'

export interface SiteDetailPanelProps {
  site: Site | null
  open: boolean
  onClose: () => void
  onCreateTicket: () => void
}

type SiteDetailTab = 'performance' | 'faultHistory' | 'tickets'

const TABS: TabItem<SiteDetailTab>[] = [
  { value: 'performance', label: 'Performance' },
  { value: 'faultHistory', label: 'Fault history' },
  { value: 'tickets', label: 'Tickets' },
]

const INCIDENT_STATUS_TONE: Record<IncidentStatus, StatusTone> = {
  Open: 'warning',
  Resolved: 'success',
}

function formatSiteTimestamp(iso: string): string {
  const date = new Date(iso)
  const datePart = date.toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    timeZone: 'Africa/Lagos',
  })
  const timePart = date.toLocaleTimeString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Africa/Lagos',
  })
  return `${datePart} · ${timePart} WAT`
}

function IncidentRow({ incident }: { incident: Incident }) {
  return (
    <div className="flex gap-3">
      <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="font-semibold text-text">{incident.title}</p>
        <p className="mt-1 text-sm text-text-muted">{incident.description}</p>
        <p className="mt-1 text-sm text-text-muted">{incident.impactNote}</p>
        <div className="mt-2">
          <StatusBadge label={incident.status} tone={INCIDENT_STATUS_TONE[incident.status]} />
        </div>
        <p className="mt-2 text-xs text-text-muted">
          {incident.id} · {formatSiteTimestamp(incident.occurredAt)} · {incident.owner}
        </p>
        {incident.linkedTicketId && (
          <Link
            to={ticketDetailPath(incident.linkedTicketId)}
            className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-primary-700 hover:underline"
          >
            Open linked ticket
            <ArrowRightIcon className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        )}
      </div>
    </div>
  )
}

export function SiteDetailPanel({ site, open, onClose, onCreateTicket }: SiteDetailPanelProps) {
  const [tab, setTab] = useState<SiteDetailTab>('performance')

  const incidentsQuery = useIncidents()
  const ticketsQuery = useTickets()

  if (!site) return null

  const siteIncidents = (incidentsQuery.data ?? []).filter(
    (incident) => incident.siteCode === site.code
  )
  const siteTickets = (ticketsQuery.data ?? []).filter((ticket) => ticket.siteCode === site.code)
  const remainingLitres = Math.round((site.dieselPercent / 100) * site.dieselTankCapacityLitres)

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={`${site.code} · ${site.region.toUpperCase()}`}
      title={site.name}
      headerActions={
        <Button icon={PlusIcon} size="sm" onClick={onCreateTicket}>
          Create ticket
        </Button>
      }
      subtitle={
        <div className="flex flex-col gap-2">
          <span className="inline-flex items-center gap-1.5">
            <MapPinIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
            {site.region}, Nigeria · Customer site
          </span>
          <StatusBadge label={site.status} tone={getSiteStatusTone(site.status)} />
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <Tabs aria-label="Site detail" items={TABS} value={tab} onChange={setTab} />

        {tab === 'performance' && (
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-medium text-text-muted">Power consumption</p>
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
                    <BoltIcon className="h-5 w-5" aria-hidden="true" />
                  </span>
                </div>
                <p className="mt-2 text-2xl font-bold text-text">
                  {site.energyKwh.toLocaleString()}{' '}
                  <span className="text-base font-medium text-text-muted">kWh</span>
                </p>
                <p className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-primary-700">
                  <ArrowTrendingUpIcon className="h-4 w-4" aria-hidden="true" />
                  Estimated
                </p>
                <p className="mt-1 text-xs text-text-muted">Selected reporting period</p>
              </div>

              <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-medium text-text-muted">Diesel remaining</p>
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
                    <FuelIcon className="h-5 w-5" aria-hidden="true" />
                  </span>
                </div>
                <p className="mt-2 text-2xl font-bold text-text">
                  {remainingLitres.toLocaleString()}{' '}
                  <span className="text-base font-medium text-text-muted">L</span>
                </p>
                <p
                  className={`mt-1 inline-flex items-center gap-1 text-sm font-medium ${site.dieselPercent < 25 ? 'text-warning-700' : 'text-primary-700'}`}
                >
                  {site.dieselPercent < 25 ? (
                    <ExclamationTriangleIcon className="h-4 w-4" aria-hidden="true" />
                  ) : (
                    <ArrowTrendingUpIcon className="h-4 w-4" aria-hidden="true" />
                  )}
                  {site.dieselPercent}% of {site.dieselTankCapacityLitres.toLocaleString()} L tank
                </p>
                <p className="mt-1 text-xs text-text-muted">Latest recorded value</p>
              </div>

              <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-medium text-text-muted">Power availability</p>
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
                    <CheckCircleIcon className="h-5 w-5" aria-hidden="true" />
                  </span>
                </div>
                <p className="mt-2 text-2xl font-bold text-text">
                  {site.powerAvailabilityPercent.toFixed(2)}
                  <span className="text-base font-medium text-text-muted">%</span>
                </p>
                <p className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-primary-700">
                  <ArrowTrendingUpIcon className="h-4 w-4" aria-hidden="true" />
                  Reported service history
                </p>
                <p className="mt-1 text-xs text-text-muted">
                  {new Date(site.updatedAt).toLocaleDateString('en-GB', {
                    day: '2-digit',
                    month: 'short',
                    year: 'numeric',
                    timeZone: 'Africa/Lagos',
                  })}
                </p>
              </div>

              <div className="min-w-0 rounded-lg border border-border bg-surface p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-medium text-text-muted">Radio availability</p>
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-600">
                    <SignalIcon className="h-5 w-5" aria-hidden="true" />
                  </span>
                </div>
                <p className="mt-2 text-2xl font-bold text-text">
                  {site.radioAvailabilityPercent == null ? (
                    '—'
                  ) : (
                    <>
                      {site.radioAvailabilityPercent.toFixed(2)}
                      <span className="text-base font-medium text-text-muted">%</span>
                    </>
                  )}
                </p>
                <p className="mt-1 inline-flex items-center gap-1 text-sm font-medium text-primary-700">
                  <ArrowTrendingUpIcon className="h-4 w-4" aria-hidden="true" />
                  {site.radioAvailabilityPercent == null
                    ? 'Awaiting data'
                    : 'Customer network report'}
                </p>
                <p className="mt-1 text-xs text-text-muted">
                  Updated {formatSiteTimestamp(site.updatedAt)}
                </p>
              </div>
            </div>

            <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
              <h3 className="text-base font-semibold text-text">Site details</h3>
              <dl className="mt-4 divide-y divide-border">
                <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <dt className="text-sm text-text-muted">Current power source</dt>
                  <dd className="text-sm font-medium text-text sm:text-right">
                    {site.powerSource}
                  </dd>
                </div>
                <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <dt className="text-sm text-text-muted">Last power observation</dt>
                  <dd className="text-sm font-medium text-text sm:text-right">
                    {formatSiteTimestamp(site.updatedAt)}
                  </dd>
                </div>
              </dl>
            </div>
          </div>
        )}

        {tab === 'faultHistory' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Fault & restoration history</h3>
            {incidentsQuery.isLoading ? (
              <p className="mt-3 text-sm text-text-muted">Loading...</p>
            ) : siteIncidents.length === 0 ? (
              <p className="mt-3 text-sm text-text-muted">No faults recorded for this site.</p>
            ) : (
              <div className="mt-4 flex flex-col gap-5">
                {siteIncidents.map((incident) => (
                  <IncidentRow key={incident.id} incident={incident} />
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'tickets' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Support requests for this site</h3>
            {ticketsQuery.isLoading ? (
              <p className="mt-3 text-sm text-text-muted">Loading...</p>
            ) : siteTickets.length === 0 ? (
              <p className="mt-3 text-sm text-text-muted">No tickets yet for this site.</p>
            ) : (
              <div className="mt-4 flex flex-col divide-y divide-border">
                {siteTickets.map((ticket) => (
                  <div key={ticket.id} className="flex items-start justify-between gap-3 py-3">
                    <div className="min-w-0">
                      <p className="text-xs font-medium text-primary-600">{ticket.id}</p>
                      <p className="font-medium text-text">{ticket.subject}</p>
                      <p className="mt-1 text-xs text-text-muted">
                        {formatDateTimeWAT(ticket.createdAt)}
                      </p>
                    </div>
                    <StatusBadge
                      label={ticket.status}
                      tone={getTicketStatusTone(ticket.status)}
                      className="shrink-0"
                    />
                  </div>
                ))}
              </div>
            )}
            <Button icon={PlusIcon} size="sm" className="mt-4" onClick={onCreateTicket}>
              Create ticket
            </Button>
          </div>
        )}
      </div>
    </SidePanel>
  )
}
