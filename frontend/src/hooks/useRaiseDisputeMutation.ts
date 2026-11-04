import { useMutation, useQueryClient } from '@tanstack/react-query'

import { raiseDispute } from '../services'
import type { RaiseDisputeInput } from '../types'
import { DISPUTE_SUMMARY_QUERY_KEY } from './useDisputeSummary'
import { DISPUTES_QUERY_KEY } from './useDisputes'
import { SITE_OVERVIEW_QUERY_KEY } from './useSiteOverview'

export function useRaiseDisputeMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RaiseDisputeInput) => raiseDispute(input),
    onSuccess: (_dispute, input) => {
      queryClient.invalidateQueries({ queryKey: DISPUTES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: DISPUTE_SUMMARY_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...SITE_OVERVIEW_QUERY_KEY, input.siteId] })
    },
  })
}
