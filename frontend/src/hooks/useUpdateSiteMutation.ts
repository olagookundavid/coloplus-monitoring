import { useMutation, useQueryClient } from '@tanstack/react-query'

import { updateSiteProfile } from '../services'
import type { SiteProfileInput } from '../types'
import { ADMIN_SITES_QUERY_KEY } from './useAdminSites'
import { SITE_PROFILES_QUERY_KEY } from './useSiteProfiles'

export interface UpdateSiteInput {
  id: string
  input: SiteProfileInput
}

export function useUpdateSiteMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: UpdateSiteInput) => updateSiteProfile(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SITE_PROFILES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ADMIN_SITES_QUERY_KEY })
    },
  })
}
