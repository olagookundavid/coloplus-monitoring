import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import {
  CONDITION_FILTER_OPTIONS,
  DashboardFilterBar,
  DashboardPageHeader,
  NO_SITES,
  periodToDateRange,
  SITE_COLUMNS,
  SiteAttentionBanner,
  SiteDetailPanel,
} from '../../components/dashboard'
import { DataTable, Select } from '../../components/ui'
import { ROUTES } from '../../constants'
import { usePortfolioOverview } from '../../hooks'
import type { PortfolioSite } from '../../types'
import { useDashboardFilters } from './CustomerLayout'

export function Sites() {
  const { region, setRegion, period, setPeriod, openCreateTicket } = useDashboardFilters()
  const [conditionFilter, setConditionFilter] = useState<
    PortfolioSite['condition'] | 'All conditions'
  >('All conditions')
  const { id: selectedSiteId } = useParams<{ id?: string }>()
  const navigate = useNavigate()

  const sitesQuery = usePortfolioOverview({ region, ...periodToDateRange(period) })
  const sites = sitesQuery.data?.sites ?? NO_SITES
  const selectedSite = selectedSiteId
    ? (sites.find((site) => site.id === selectedSiteId) ?? null)
    : null

  const filteredSites = useMemo(() => {
    if (conditionFilter === 'All conditions') return sites
    return sites.filter((site) => site.condition === conditionFilter)
  }, [sites, conditionFilter])

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
        subtitle="Explore site condition, incidents, disputes and commissioning status."
      />

      <DashboardFilterBar
        region={region}
        onRegionChange={setRegion}
        regionOptions={sitesQuery.data?.regions ?? []}
        period={period}
        onPeriodChange={setPeriod}
        isoTimestamp={sitesQuery.data?.generatedAt}
      />

      <SiteAttentionBanner sites={sites} />

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
          onRowClick={(site) => navigate(`${ROUTES.CUSTOMER_SITES}/${site.id}`)}
          searchPlaceholder="Search sites or ID..."
          emptyMessage="No sites match this filter."
          headerActions={
            <Select
              value={conditionFilter}
              onChange={(event) =>
                setConditionFilter(
                  event.target.value as PortfolioSite['condition'] | 'All conditions'
                )
              }
              aria-label="Filter by condition"
              className="w-48"
            >
              {CONDITION_FILTER_OPTIONS.map((option) => (
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
        onCreateTicket={() => openCreateTicket(selectedSite?.id)}
      />
    </div>
  )
}
