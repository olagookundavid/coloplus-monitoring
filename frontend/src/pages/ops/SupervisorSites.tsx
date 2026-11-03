import { useState } from 'react'
import { MapPinIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { SkeletonListItems } from '../../components/skeletons'
import { DataTable, SidePanel, StatusBadge, type Column, type StatusTone } from '../../components/ui'
import { useAdminSites, useSiteOverview } from '../../hooks'
import type { AdminSite } from '../../types'
import { formatDateTimeWAT, severityLabel } from '../../utils'

const NO_SITES: AdminSite[] = []

const CONDITION: Record<string, { label: string; tone: StatusTone }> = {
  available: { label: 'Available', tone: 'success' },
  outage_reported: { label: 'Outage reported', tone: 'danger' },
  awaiting_update: { label: 'Awaiting update', tone: 'warning' },
}

function SiteDetailBody({ site }: { site: AdminSite }) {
  const overviewQuery = useSiteOverview(site.id)
  const overview = overviewQuery.data
  const condition = overview ? CONDITION[overview.condition] : undefined

  return (
    <div className="flex flex-col gap-5">
      <dl className="divide-y divide-border rounded-lg border border-border">
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Airtel code</dt>
          <dd className="font-medium text-text">{site.airtelCode}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Coloplus code</dt>
          <dd className="font-medium text-text">{site.coloplusCode}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Type</dt>
          <dd className="font-medium text-text">{site.siteType}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Region / State</dt>
          <dd className="font-medium text-text">
            {site.region} · {site.state}
          </dd>
        </div>
      </dl>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          Live condition
        </p>
        {overviewQuery.isLoading ? (
          <SkeletonListItems count={1} withBadge={false} />
        ) : condition ? (
          <div className="flex items-center justify-between rounded-lg border border-border bg-surface p-4">
            <StatusBadge label={condition.label} tone={condition.tone} />
            {overview?.asOf && (
              <span className="text-xs text-text-muted">as of {formatDateTimeWAT(overview.asOf)}</span>
            )}
          </div>
        ) : (
          <p className="text-sm text-text-muted">No report yet.</p>
        )}
      </div>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          Open incidents ({overview?.openIncidents ?? 0})
        </p>
        {overviewQuery.isLoading ? (
          <SkeletonListItems count={2} />
        ) : (overview?.incidents.length ?? 0) === 0 ? (
          <p className="text-sm text-text-muted">No open incidents at this site.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {overview?.incidents.map((incident) => {
              const { label, tone } = severityLabel(incident.severity)
              return (
                <li
                  key={incident.id}
                  className="rounded-lg border border-border bg-surface p-3 text-sm"
                >
                  <div className="flex items-center justify-between gap-2">
                    <StatusBadge label={label} tone={tone} />
                    <span className="text-xs text-text-muted">{incident.ref}</span>
                  </div>
                  <p className="mt-1.5 font-medium text-text">{incident.title}</p>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

export function SupervisorSites() {
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)
  const selectedSite = sites.find((site) => site.id === selectedSiteId) ?? null

  const columns: Column<AdminSite>[] = [
    { header: 'Site', accessor: (site) => site.airtelCode },
    {
      header: 'Name and type',
      accessor: (site) => site.name,
      cell: (site) => (
        <div>
          <p className="font-medium text-text">{site.name}</p>
          <p className="text-xs text-text-muted">{site.siteType}</p>
        </div>
      ),
    },
    { header: 'Region', accessor: (site) => site.region },
    { header: 'State', accessor: (site) => site.state },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Sites"
        subtitle="Every site in your scope. Open one for its live condition and open incidents."
      />

      <DataTable
        data={sites}
        columns={columns}
        getRowId={(site) => site.id}
        loading={sitesQuery.isLoading}
        onRowClick={(site) => setSelectedSiteId(site.id)}
        searchPlaceholder="Search site code or name..."
        emptyMessage="No sites in your scope yet."
      />

      <SidePanel
        open={Boolean(selectedSite)}
        onClose={() => setSelectedSiteId(null)}
        eyebrow={selectedSite?.airtelCode}
        title={selectedSite?.name ?? ''}
        subtitle={
          selectedSite && (
            <span className="inline-flex items-center gap-1.5">
              <MapPinIcon className="h-4 w-4" aria-hidden="true" />
              {selectedSite.region}, {selectedSite.state}
            </span>
          )
        }
      >
        {selectedSite && <SiteDetailBody site={selectedSite} />}
      </SidePanel>
    </div>
  )
}
