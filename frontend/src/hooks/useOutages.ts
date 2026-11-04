import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { fetchOutages } from '../services'
import type { OutageListParams } from '../types'

export const OUTAGES_QUERY_KEY = ['outages']

export function useOutages(params: OutageListParams, enabled = true) {
  return useQuery({
    queryKey: [...OUTAGES_QUERY_KEY, params],
    queryFn: () => fetchOutages(params),
    placeholderData: keepPreviousData,
    enabled,
  })
}
