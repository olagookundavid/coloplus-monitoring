import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createSiteProfile } from '../services'
import type { SiteProfileInput } from '../types'
import { ADMIN_SITES_QUERY_KEY } from './useAdminSites'
import { SITE_PROFILES_QUERY_KEY } from './useSiteProfiles'

export function useCreateSiteMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: SiteProfileInput) => createSiteProfile(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SITE_PROFILES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ADMIN_SITES_QUERY_KEY })
    },
  })
}
