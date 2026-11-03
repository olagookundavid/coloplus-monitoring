import { useMutation, useQueryClient } from '@tanstack/react-query'

import { reopenTicket } from '../services'
import { TICKET_DETAIL_QUERY_KEY } from './useTicketDetail'
import { TICKET_SUMMARY_QUERY_KEY } from './useTicketSummary'
import { TICKETS_QUERY_KEY } from './useTickets'

export function useReopenTicketMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) => reopenTicket(id, note),
    onSuccess: (_ticket, { id }) => {
      queryClient.invalidateQueries({ queryKey: TICKETS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...TICKET_DETAIL_QUERY_KEY, id] })
      queryClient.invalidateQueries({ queryKey: TICKET_SUMMARY_QUERY_KEY })
    },
  })
}
