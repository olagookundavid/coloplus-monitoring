import { useMutation, useQueryClient } from '@tanstack/react-query'

import { transitionAcceptance } from '../services'
import type { TransitionAcceptanceInput } from '../types'
import { ACCEPTANCES_QUERY_KEY } from './useAcceptances'
import { SITE_ACCEPTANCE_QUERY_KEY } from './useSiteAcceptance'
import { SITE_OVERVIEW_QUERY_KEY } from './useSiteOverview'

export function useTransitionAcceptanceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ siteId, input }: { siteId: string; input: TransitionAcceptanceInput }) =>
      transitionAcceptance(siteId, input),
    onSuccess: (_acceptance, { siteId }) => {
      queryClient.invalidateQueries({ queryKey: ACCEPTANCES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...SITE_ACCEPTANCE_QUERY_KEY, siteId] })
      queryClient.invalidateQueries({ queryKey: [...SITE_OVERVIEW_QUERY_KEY, siteId] })
    },
  })
}
