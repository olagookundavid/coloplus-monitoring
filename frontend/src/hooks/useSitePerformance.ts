import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { getSitePerformance, type PortfolioQueryParams } from '../services'

export function useSitePerformance(params: PortfolioQueryParams) {
  return useQuery({
    queryKey: ['site-performance', params],
    queryFn: () => getSitePerformance(params),
    placeholderData: keepPreviousData,
  })
}
