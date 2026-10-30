import { useQuery } from '@tanstack/react-query'

import { fetchPendingApprovals } from '../services'

export function usePendingApprovals() {
  return useQuery({
    queryKey: ['pending-approvals'],
    queryFn: () => fetchPendingApprovals(),
  })
}
