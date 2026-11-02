import { useMutation } from '@tanstack/react-query'

import { regenerateRecoveryCodes } from '../services'
import type { EnableMfaRequest } from '../types'

export function useRegenerateRecoveryCodesMutation() {
  return useMutation({
    mutationFn: (request: EnableMfaRequest) => regenerateRecoveryCodes(request),
  })
}
