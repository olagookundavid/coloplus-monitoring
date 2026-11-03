import { useQuery } from '@tanstack/react-query'

import { fetchSiteAcceptance } from '../services'

export const SITE_ACCEPTANCE_QUERY_KEY = ['site-acceptance']

export function useSiteAcceptance(siteId: string | null | undefined) {
  return useQuery({
    queryKey: [...SITE_ACCEPTANCE_QUERY_KEY, siteId],
    queryFn: () => fetchSiteAcceptance(siteId as string),
    enabled: Boolean(siteId),
  })
}
