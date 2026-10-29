import { useMutation } from '@tanstack/react-query'

import { requestPasswordReset } from '../services'
import type { ForgotPasswordRequest } from '../types'

export function useForgotPasswordMutation() {
  return useMutation({
    mutationFn: (request: ForgotPasswordRequest) => requestPasswordReset(request),
  })
}
