import { useMutation } from '@tanstack/react-query'

import { resendInvite } from '../services'

export function useResendInviteMutation() {
  return useMutation({
    mutationFn: (id: string) => resendInvite(id),
  })
}
