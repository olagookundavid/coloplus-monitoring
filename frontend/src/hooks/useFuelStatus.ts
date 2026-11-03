import { useQuery } from '@tanstack/react-query'

import { fetchFuelStatus } from '../services'

export const FUEL_QUERY_KEY = ['fuel']

export function useFuelStatus(siteId: string | undefined) {
  return useQuery({
    queryKey: [...FUEL_QUERY_KEY, siteId],
    queryFn: () => fetchFuelStatus(siteId as string),
    enabled: Boolean(siteId),
  })
}
