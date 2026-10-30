import { useMutation, useQueryClient } from '@tanstack/react-query'

import { createTicket } from '../services'
import type { CreateTicketRequest } from '../types'

export function useCreateTicketMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (request: CreateTicketRequest) => createTicket(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
    },
  })
}
