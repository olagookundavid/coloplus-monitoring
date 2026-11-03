import { useQuery } from '@tanstack/react-query'

import { fetchTickets } from '../services'
import type { TicketFilter } from '../services'

export const TICKETS_QUERY_KEY = ['tickets']

export function useTickets(filter: TicketFilter = {}) {
  return useQuery({
    queryKey: [...TICKETS_QUERY_KEY, filter],
    queryFn: () => fetchTickets(filter),
  })
}
