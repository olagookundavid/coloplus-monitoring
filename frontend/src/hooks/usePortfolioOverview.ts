import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { getPortfolioOverview, type PortfolioQueryParams } from '../services'

export function usePortfolioOverview(params: PortfolioQueryParams) {
  return useQuery({
    queryKey: ['portfolio-overview', params],
    queryFn: () => getPortfolioOverview(params),
    placeholderData: keepPreviousData,
  })
}
