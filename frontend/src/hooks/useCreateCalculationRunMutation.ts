import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createCalculationRun } from '../services'
import type { CreateCalculationRunInput } from '../types'
import { CALCULATION_RUNS_QUERY_KEY } from './useCalculationRuns'

export function useCreateCalculationRunMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCalculationRunInput) => createCalculationRun(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CALCULATION_RUNS_QUERY_KEY })
    },
  })
}
