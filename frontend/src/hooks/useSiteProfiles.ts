import { useQuery } from '@tanstack/react-query'

import { fetchSiteProfiles } from '../services'

export const SITE_PROFILES_QUERY_KEY = ['site-profiles']

export function useSiteProfiles() {
  return useQuery({
    queryKey: SITE_PROFILES_QUERY_KEY,
    queryFn: fetchSiteProfiles,
  })
}
