import { useMutation, useQueryClient } from '@tanstack/react-query'

import { raiseDispute } from '../services'
import type { RaiseDisputeInput } from '../types'
import { DISPUTE_SUMMARY_QUERY_KEY } from './useDisputeSummary'
import { DISPUTES_QUERY_KEY } from './useDisputes'

export function useRaiseDisputeMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RaiseDisputeInput) => raiseDispute(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: DISPUTES_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: DISPUTE_SUMMARY_QUERY_KEY })
    },
  })
}
