import { useMemo } from 'react'

import { useIncidentCorrections } from './useIncidentCorrections'
import { useMyReports } from './useMyReports'
import { useMySites } from './useMySites'
import type { CorrectionListItem, DailyReport, IncidentCorrection, MySiteRow } from '../types'
import { buildCorrectionList } from '../utils'

const NO_REPORTS: DailyReport[] = []
const NO_CORRECTIONS: IncidentCorrection[] = []
const NO_SITES: MySiteRow[] = []
const NO_ITEMS: CorrectionListItem[] = []

// The Corrections page's data: the reporter's own reports (any status, for
// both the returned-report and report-correction cases) and incident
// corrections, fetched in parallel -- no dependent waterfall -- then merged
// client-side into one tab-bucketed list.
export function useCorrections() {
  const reportsQuery = useMyReports({ limit: 200 })
  const correctionsQuery = useIncidentCorrections()
  const sitesQuery = useMySites()

  const sites = sitesQuery.data?.sites ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.siteId, site])), [sites])

  const items = useMemo(
    () => buildCorrectionList(reportsQuery.data?.reports ?? NO_REPORTS, correctionsQuery.data ?? NO_CORRECTIONS),
    [reportsQuery.data, correctionsQuery.data]
  )

  return {
    items: items.length ? items : NO_ITEMS,
    sites,
    sitesById,
    isLoading: reportsQuery.isLoading || correctionsQuery.isLoading || sitesQuery.isLoading,
  }
}
