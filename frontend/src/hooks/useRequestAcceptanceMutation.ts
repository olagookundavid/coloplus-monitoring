import { useMutation, useQueryClient } from '@tanstack/react-query'

import { requestAcceptance } from '../services'
import type { RequestAcceptanceInput } from '../services'
import { ACCEPTANCES_QUERY_KEY } from './useAcceptances'
import { SITE_ACCEPTANCE_QUERY_KEY } from './useSiteAcceptance'

export function useRequestAcceptanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ siteId, input }: { siteId: string; input?: RequestAcceptanceInput }) =>
      requestAcceptance(siteId, input),
    onSuccess: (_acceptance, { siteId }) => {
      queryClient.invalidateQueries({ queryKey: ACCEPTANCES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...SITE_ACCEPTANCE_QUERY_KEY, siteId] })
    },
  })
}
