import { useMutation } from '@tanstack/react-query'

import { enableMfa } from '../services'
import type { EnableMfaRequest } from '../types'

export function useEnableMfaMutation() {
  return useMutation({
    mutationFn: (request: EnableMfaRequest) => enableMfa(request),
  })
}
