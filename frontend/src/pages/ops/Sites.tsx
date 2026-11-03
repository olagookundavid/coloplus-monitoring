import { useMemo, useState } from 'react'
import { PencilSquareIcon, PlusIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, SiteFormPanel } from '../../components/dashboard'
import {
  Button,
  DataTable,
  RowActions,
  StatusBadge,
  type Column,
  type RowAction,
} from '../../components/ui'
import { useSiteProfiles } from '../../hooks'
import { POWER_SOURCE_LABELS, type SiteProfile } from '../../types'
import { siteStatusLabel } from '../../utils'

const NO_SITES: SiteProfile[] = []

export function Sites() {
  const sitesQuery = useSiteProfiles()
  const sites = sitesQuery.data ?? NO_SITES

  const [createOpen, setCreateOpen] = useState(false)
  const [editSite, setEditSite] = useState<SiteProfile | null>(null)

  const columns: Column<SiteProfile>[] = useMemo(
    () => [
      { header: 'Site', accessor: (site) => site.airtelCode },
      {
        header: 'Name and type',
        accessor: (site) => site.name,
        cell: (site) => (
          <div>
            <p className="font-medium text-text">{site.name}</p>
            <p className="text-xs text-text-muted">{site.siteType || '—'}</p>
          </div>
        ),
      },
      {
        header: 'Region / State',
        accessor: (site) => `${site.region} ${site.state}`,
        cell: (site) => (
          <span>
            {site.region || '—'} · {site.state}
          </span>
        ),
      },
      {
        header: 'Power sources',
        accessor: (site) => site.powerSources.join(', '),
        cell: (site) => site.powerSources.map((s) => POWER_SOURCE_LABELS[s]).join(', ') || '—',
      },
      {
        header: 'Status',
        accessor: (site) => site.status,
        cell: (site) => {
          const { label, tone } = siteStatusLabel(site.status)
          return <StatusBadge label={label} tone={tone} />
        },
      },
      {
        header: 'Actions',
        accessor: () => null,
        align: 'right',
        cell: (site) => {
          const actions: RowAction[] = [
            {
              label: 'Edit',
              icon: PencilSquareIcon,
              onClick: () => setEditSite(site),
            },
          ]
          return <RowActions actions={actions} label={`Actions for ${site.name}`} />
        },
      },
    ],
    []
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Sites"
        subtitle="The site master record: codes, location, power, fuel and reporting rules."
        actions={
          <Button icon={PlusIcon} onClick={() => setCreateOpen(true)}>
            New site
          </Button>
        }
      />

      <DataTable
        data={sites}
        columns={columns}
        getRowId={(site) => site.id}
        loading={sitesQuery.isLoading}
        searchPlaceholder="Search site code or name..."
        emptyMessage="No sites yet."
      />

      <SiteFormPanel open={createOpen} onClose={() => setCreateOpen(false)} />

      <SiteFormPanel
        key={editSite?.id}
        open={Boolean(editSite)}
        onClose={() => setEditSite(null)}
        site={editSite ?? undefined}
      />
    </div>
  )
}
