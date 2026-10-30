import { useEffect, useMemo, useState } from 'react'
import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'
import { useNavigate, useParams } from 'react-router-dom'

import {
  DashboardFilterBar,
  DashboardPageHeader,
  NO_SITES,
  SITE_COLUMNS,
  SiteAttentionBanner,
  SiteDetailPanel,
  STATUS_FILTER_OPTIONS,
} from '../../components/dashboard'
import { Button, DataTable, Select } from '../../components/ui'
import { ROUTES, siteDetailPath } from '../../constants'
import { useSitePerformance } from '../../hooks'
import type { SiteStatus } from '../../types'
import { useDashboardFilters } from './CustomerLayout'

export function Sites() {
  const { region, setRegion, period, setPeriod, openCreateTicket } = useDashboardFilters()
  const [statusFilter, setStatusFilter] = useState<SiteStatus | 'All statuses'>('All statuses')
  const { id: selectedSiteId } = useParams<{ id?: string }>()
  const navigate = useNavigate()

  const sitesQuery = useSitePerformance({ region, period })
  const sites = sitesQuery.data ?? NO_SITES
  const selectedSite = selectedSiteId
    ? (sites.find((site) => site.id === selectedSiteId) ?? null)
    : null

  const filteredSites = useMemo(() => {
    if (statusFilter === 'All statuses') return sites
    return sites.filter((site) => site.status === statusFilter)
  }, [sites, statusFilter])

  // A stale or mistyped site id in the URL — send the customer back to the
  // list rather than leaving the panel open with nothing to show.
  useEffect(() => {
    if (selectedSiteId && !sitesQuery.isLoading && !selectedSite) {
      navigate(ROUTES.CUSTOMER_SITES, { replace: true })
    }
  }, [selectedSiteId, sitesQuery.isLoading, selectedSite, navigate])

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Site operations"
        title="Your site portfolio"
        subtitle="Explore site performance, fuel levels and service history."
        actions={
          <Button variant="outline" icon={ArrowDownTrayIcon}>
            Export report
          </Button>
        }
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        period={period}
        onPeriodChange={setPeriod}
      />

      <SiteAttentionBanner sites={sitesQuery.data ?? NO_SITES} />

      <div>
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-text">Site performance</h2>
          <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-text-muted">
            {filteredSites.length} {filteredSites.length === 1 ? 'site' : 'sites'}
          </span>
        </div>
        <DataTable
          data={filteredSites}
          columns={SITE_COLUMNS}
          getRowId={(site) => site.id}
          loading={sitesQuery.isLoading}
          getRowHref={(site) => siteDetailPath(site.id)}
          searchPlaceholder="Search sites or ID..."
          emptyMessage="No sites match this filter."
          headerActions={
            <Select
              value={statusFilter}
              onChange={(event) =>
                setStatusFilter(event.target.value as SiteStatus | 'All statuses')
              }
              aria-label="Filter by status"
              className="w-44"
            >
              {STATUS_FILTER_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </Select>
          }
        />
      </div>

      <SiteDetailPanel
        site={selectedSite}
        open={Boolean(selectedSite)}
        onClose={() => navigate(ROUTES.CUSTOMER_SITES)}
        onCreateTicket={() => openCreateTicket(selectedSite?.code)}
      />
    </div>
  )
}
