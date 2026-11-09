import { useMutation, useQueryClient } from '@tanstack/react-query'

import { transitionTicket } from '../services'
import type { TransitionTicketInput } from '../services'
import { SITE_OVERVIEW_QUERY_KEY } from './useSiteOverview'
import { TICKET_DETAIL_QUERY_KEY } from './useTicketDetail'
import { TICKET_SUMMARY_QUERY_KEY } from './useTicketSummary'
import { TICKETS_QUERY_KEY } from './useTickets'

// Coloplus's general transition (start work, resolve, take a resolved
// ticket back into work) -- distinct from useReopenTicketMutation, which is
// the customer's own reopen and always sends status "open".
export function useTransitionTicketMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: TransitionTicketInput }) =>
      transitionTicket(id, input),
    onSuccess: (ticket, { id }) => {
      queryClient.invalidateQueries({ queryKey: TICKETS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...TICKET_DETAIL_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: TICKET_SUMMARY_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...SITE_OVERVIEW_QUERY_KEY, ticket.siteId] })
    },
  })
}
