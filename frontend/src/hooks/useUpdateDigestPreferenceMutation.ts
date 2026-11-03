import { useMutation, useQueryClient } from '@tanstack/react-query'

import { updateDigestPreference } from '../services'
import { DIGEST_PREFERENCE_QUERY_KEY } from './useDigestPreference'

export function useUpdateDigestPreferenceMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (weekly: boolean) => updateDigestPreference(weekly),
    onSuccess: (preference) => {
      queryClient.setQueryData(DIGEST_PREFERENCE_QUERY_KEY, preference)
    },
  })
}
