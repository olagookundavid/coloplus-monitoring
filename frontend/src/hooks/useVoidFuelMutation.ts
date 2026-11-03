import { useMutation, useQueryClient } from '@tanstack/react-query'

import { voidFuelEntry } from '../services'
import { FUEL_QUERY_KEY } from './useFuelStatus'

export interface VoidFuelInput {
  id: string
  reason: string
}

export function useVoidFuelMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: VoidFuelInput) => voidFuelEntry(id, reason),
    onSuccess: (entry) => {
      queryClient.invalidateQueries({ queryKey: [...FUEL_QUERY_KEY, entry.siteId] })
    },
  })
}
