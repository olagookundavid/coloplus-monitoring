import { useMutation, useQueryClient } from '@tanstack/react-query'

import { recordFuel } from '../services'
import type { RecordFuelInput } from '../types'
import { FUEL_QUERY_KEY } from './useFuelStatus'

export function useRecordFuelMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RecordFuelInput) => recordFuel(input),
    onSuccess: (_entry, input) => {
      queryClient.invalidateQueries({ queryKey: [...FUEL_QUERY_KEY, input.siteId] })
    },
  })
}
