import { useMutation } from '@tanstack/react-query'

import { setPassword } from '../services'
import type { SetPasswordRequest } from '../types'

export function useSetPasswordMutation() {
  return useMutation({
    mutationFn: (request: SetPasswordRequest) => setPassword(request),
  })
}
