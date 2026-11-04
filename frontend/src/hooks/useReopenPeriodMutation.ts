import { useMutation, useQueryClient } from '@tanstack/react-query'

import { reopenPeriod } from '../services'
import { PERIODS_QUERY_KEY } from './usePeriods'

export interface ReopenPeriodInput {
  month: string
  reason: string
}

export function useReopenPeriodMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ month, reason }: ReopenPeriodInput) => reopenPeriod(month, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PERIODS_QUERY_KEY })
    },
  })
}
