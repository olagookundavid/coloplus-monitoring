import { useQuery } from '@tanstack/react-query'

import { fetchDisputeSummary } from '../services'

export const DISPUTE_SUMMARY_QUERY_KEY = ['dispute-summary']

export function useDisputeSummary() {
  return useQuery({
    queryKey: DISPUTE_SUMMARY_QUERY_KEY,
    queryFn: fetchDisputeSummary,
  })
}
