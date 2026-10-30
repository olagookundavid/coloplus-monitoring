import { useQuery } from '@tanstack/react-query'

import { getIncidents } from '../services'

export function useIncidents() {
  return useQuery({
    queryKey: ['incidents'],
    queryFn: () => getIncidents(),
  })
}
