import { useMutation, useQueryClient } from '@tanstack/react-query'

import { linkTicketIncident } from '../services'
import { TICKETS_QUERY_KEY } from './useTickets'

export interface LinkTicketIncidentInput {
  ticketId: string
  incidentId: string
}

export function useLinkTicketIncidentMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ticketId, incidentId }: LinkTicketIncidentInput) =>
      linkTicketIncident(ticketId, incidentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TICKETS_QUERY_KEY })
    },
  })
}
