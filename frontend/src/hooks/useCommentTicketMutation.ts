import { useMutation, useQueryClient } from '@tanstack/react-query'

import { commentTicket } from '../services'
import { TICKET_DETAIL_QUERY_KEY } from './useTicketDetail'

export function useCommentTicketMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body, internal }: { id: string; body: string; internal?: boolean }) =>
      commentTicket(id, body, internal),
    onSuccess: (_event, { id }) => {
      queryClient.invalidateQueries({ queryKey: [...TICKET_DETAIL_QUERY_KEY, id] })
    },
  })
}
