import { useQuery } from '@tanstack/react-query'

import { fetchDisputeDetail } from '../services'

export const DISPUTE_DETAIL_QUERY_KEY = ['dispute-detail']

export function useDisputeDetail(id: string | null | undefined) {
  return useQuery({
    queryKey: [...DISPUTE_DETAIL_QUERY_KEY, id],
    queryFn: () => fetchDisputeDetail(id as string),
    enabled: Boolean(id),
  })
}
