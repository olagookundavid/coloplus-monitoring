import { useMutation, useQueryClient } from '@tanstack/react-query'

import { commentDispute } from '../services'
import { DISPUTE_DETAIL_QUERY_KEY } from './useDisputeDetail'

export function useCommentDisputeMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body, internal }: { id: string; body: string; internal?: boolean }) =>
      commentDispute(id, body, internal),
    onSuccess: (_event, { id }) => {
      queryClient.invalidateQueries({ queryKey: [...DISPUTE_DETAIL_QUERY_KEY, id] })
    },
  })
}
