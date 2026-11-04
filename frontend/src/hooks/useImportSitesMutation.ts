import { useMutation, useQueryClient } from '@tanstack/react-query'

import { importSitesCsv } from '../services'
import type { ImportSitesInput } from '../services'
import { ADMIN_SITES_QUERY_KEY } from './useAdminSites'
import { SITE_PROFILES_QUERY_KEY } from './useSiteProfiles'

export function useImportSitesMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: ImportSitesInput) => importSitesCsv(input),
    onSuccess: (_result, variables) => {
      // A dry run writes nothing -- nothing to invalidate.
      if (variables.dryRun) return
      queryClient.invalidateQueries({ queryKey: ADMIN_SITES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: SITE_PROFILES_QUERY_KEY })
    },
  })
}
