import { useMutation } from '@tanstack/react-query'

import { setupMfa } from '../services'

export function useMfaSetupMutation() {
  return useMutation({
    mutationFn: () => setupMfa(),
  })
}
