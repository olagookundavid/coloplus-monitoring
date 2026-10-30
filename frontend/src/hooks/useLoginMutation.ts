import { useMutation } from '@tanstack/react-query'

import { login } from '../services'
import type { LoginRequest } from '../types'

export function useLoginMutation() {
  return useMutation({
    mutationFn: (request: LoginRequest) => login(request),
  })
}
