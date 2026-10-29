import { useMutation } from '@tanstack/react-query'

import { acceptInvite } from '../services'
import type { AcceptInviteRequest } from '../types'

export function useAcceptInviteMutation() {
  return useMutation({
    mutationFn: (request: AcceptInviteRequest) => acceptInvite(request),
  })
}
