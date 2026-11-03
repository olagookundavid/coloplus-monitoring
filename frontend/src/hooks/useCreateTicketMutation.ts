import { useMutation, useQueryClient } from '@tanstack/react-query'

import { raiseTicket } from '../services'
import type { RaiseTicketInput } from '../types'
import { TICKET_SUMMARY_QUERY_KEY } from './useTicketSummary'
import { TICKETS_QUERY_KEY } from './useTickets'

export function useCreateTicketMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RaiseTicketInput) => raiseTicket(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TICKETS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: TICKET_SUMMARY_QUERY_KEY })
    },
  })
}
