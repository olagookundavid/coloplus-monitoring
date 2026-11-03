import { useMemo, useState } from 'react'

import { DashboardPageHeader } from '../../components/dashboard'
import { SkeletonListItems } from '../../components/skeletons'
import { SidePanel, StatusBadge, Tabs, type TabItem } from '../../components/ui'
import { useAdminSites, usePublishedIncidents } from '../../hooks'
import type { AdminSite, PublishedIncident } from '../../types'
import { describeIncidentStatus, formatDateTimeWAT, severityLabel } from '../../utils'

const NO_INCIDENTS: PublishedIncident[] = []
const NO_SITES: AdminSite[] = []

type TabValue = 'open' | 'closed' | 'all'

function siteLabelFor(sitesById: Map<string, AdminSite>, siteId: string): string {
  return sitesById.get(siteId)?.airtelCode ?? siteId
}

export function Incidents() {
  const incidentsQuery = usePublishedIncidents()
  const incidents = incidentsQuery.data ?? NO_INCIDENTS
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.id, site])), [sites])

  const [tab, setTab] = useState<TabValue>('open')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const counts = useMemo(() => {
    const result = { open: 0, closed: 0 }
    for (const incident of incidents) {
      if (describeIncidentStatus(incident.status).bucket === 'closed') result.closed++
      else result.open++
    }
    return result
  }, [incidents])

  const visibleIncidents = useMemo(() => {
    if (tab === 'all') return incidents
    return incidents.filter(
      (incident) => (describeIncidentStatus(incident.status).bucket === 'closed') === (tab === 'closed')
    )
  }, [incidents, tab])

  const selectedIncident = incidents.find((incident) => incident.id === selectedId) ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'open', label: `Open ${counts.open}` },
    { value: 'closed', label: `Closed ${counts.closed}` },
    { value: 'all', label: `All ${incidents.length}` },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Fault register"
        title="Incidents"
        subtitle="Published faults across your sites — times, status and the updates shared with you."
      />

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <ul className="flex flex-col gap-2">
        {incidentsQuery.isLoading ? (
          <SkeletonListItems count={5} />
        ) : visibleIncidents.length === 0 ? (
          <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
            Nothing here.
          </li>
        ) : (
          visibleIncidents.map((incident) => {
            const { label: sevLabel, tone: sevTone } = severityLabel(incident.severity)
            const { label: statusText, tone: statusTone } = describeIncidentStatus(incident.status)
            return (
              <li key={incident.id}>
                <button
                  type="button"
                  onClick={() => setSelectedId(incident.id)}
                  className="w-full rounded-lg border border-border bg-surface p-4 text-left transition-colors hover:bg-surface-subtle"
                >
                  <div className="flex items-center justify-between gap-2">
                    <StatusBadge label={sevLabel} tone={sevTone} />
                    <span className="text-xs text-text-muted">{incident.ref}</span>
                  </div>
                  <p className="mt-2 font-medium text-text">
                    {incident.title} · {siteLabelFor(sitesById, incident.siteId)}
                  </p>
                  <p className="mt-1 text-sm text-text-muted">
                    {formatDateTimeWAT(incident.startedAt)} ·{' '}
                    <StatusBadge label={statusText} tone={statusTone} />
                  </p>
                </button>
              </li>
            )
          })
        )}
      </ul>

      <SidePanel
        open={Boolean(selectedIncident)}
        onClose={() => setSelectedId(null)}
        eyebrow={selectedIncident?.ref}
        title={selectedIncident?.title ?? ''}
        subtitle={
          selectedIncident && (
            <StatusBadge
              label={describeIncidentStatus(selectedIncident.status).label}
              tone={describeIncidentStatus(selectedIncident.status).tone}
            />
          )
        }
      >
        {selectedIncident && (
          <div className="flex flex-col gap-4">
            <dl className="divide-y divide-border rounded-lg border border-border">
              <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <dt className="text-text-muted">Site</dt>
                <dd className="font-medium text-text">
                  {siteLabelFor(sitesById, selectedIncident.siteId)}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <dt className="text-text-muted">Started</dt>
                <dd className="font-medium text-text">
                  {formatDateTimeWAT(selectedIncident.startedAt)}
                </dd>
              </div>
              {selectedIncident.restoredAt && (
                <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                  <dt className="text-text-muted">Restored</dt>
                  <dd className="font-medium text-text">
                    {formatDateTimeWAT(selectedIncident.restoredAt)}
                  </dd>
                </div>
              )}
              {selectedIncident.closedAt && (
                <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                  <dt className="text-text-muted">Closed</dt>
                  <dd className="font-medium text-text">
                    {formatDateTimeWAT(selectedIncident.closedAt)}
                  </dd>
                </div>
              )}
            </dl>

            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
                Updates
              </p>
              {selectedIncident.updates.length === 0 ? (
                <p className="mt-3 text-sm text-text-muted">No updates yet.</p>
              ) : (
                <ul className="mt-3 flex flex-col gap-3">
                  {selectedIncident.updates.map((update, index) => (
                    <li key={index} className="flex gap-3 text-sm">
                      <span className="w-32 shrink-0 text-text-muted">
                        {formatDateTimeWAT(update.at)}
                      </span>
                      <span className="text-text">{update.body}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
      </SidePanel>
    </div>
  )
}
