import { useQuery } from '@tanstack/react-query'

import { fetchSiteOverview } from '../services'

export const SITE_OVERVIEW_QUERY_KEY = ['site-overview']

export function useSiteOverview(siteId: string | null | undefined) {
  return useQuery({
    queryKey: [...SITE_OVERVIEW_QUERY_KEY, siteId],
    queryFn: () => fetchSiteOverview(siteId as string),
    enabled: Boolean(siteId),
  })
}
