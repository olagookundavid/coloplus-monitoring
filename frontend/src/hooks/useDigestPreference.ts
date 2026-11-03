import { useQuery } from '@tanstack/react-query'

import { fetchDigestPreference } from '../services'

export const DIGEST_PREFERENCE_QUERY_KEY = ['digest-preference']

export function useDigestPreference(enabled = true) {
  return useQuery({
    queryKey: DIGEST_PREFERENCE_QUERY_KEY,
    queryFn: fetchDigestPreference,
    enabled,
  })
}
