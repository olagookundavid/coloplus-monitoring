import { useQuery } from '@tanstack/react-query'

import { fetchTicketSummary } from '../services'

export const TICKET_SUMMARY_QUERY_KEY = ['ticket-summary']

export function useTicketSummary() {
  return useQuery({
    queryKey: TICKET_SUMMARY_QUERY_KEY,
    queryFn: fetchTicketSummary,
  })
}
