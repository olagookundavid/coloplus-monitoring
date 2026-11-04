import { useMutation, useQueryClient } from '@tanstack/react-query'

import { closePeriod } from '../services'
import { PERIODS_QUERY_KEY } from './usePeriods'

export function useClosePeriodMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (month: string) => closePeriod(month),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PERIODS_QUERY_KEY })
    },
  })
}
