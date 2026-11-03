import { useQuery } from '@tanstack/react-query'

import { fetchTicketDetail } from '../services'

export const TICKET_DETAIL_QUERY_KEY = ['ticket-detail']

export function useTicketDetail(id: string | null | undefined) {
  return useQuery({
    queryKey: [...TICKET_DETAIL_QUERY_KEY, id],
    queryFn: () => fetchTicketDetail(id as string),
    enabled: Boolean(id),
  })
}
