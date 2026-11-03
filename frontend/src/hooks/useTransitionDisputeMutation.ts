import { useMutation, useQueryClient } from '@tanstack/react-query'

import { transitionDispute } from '../services'
import type { TransitionDisputeInput } from '../services'
import { DISPUTE_DETAIL_QUERY_KEY } from './useDisputeDetail'
import { DISPUTE_SUMMARY_QUERY_KEY } from './useDisputeSummary'
import { DISPUTES_QUERY_KEY } from './useDisputes'

export function useTransitionDisputeMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TransitionDisputeInput }) =>
      transitionDispute(id, input),
    onSuccess: (_dispute, { id }) => {
      queryClient.invalidateQueries({ queryKey: DISPUTES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...DISPUTE_DETAIL_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: DISPUTE_SUMMARY_QUERY_KEY })
    },
  })
}
