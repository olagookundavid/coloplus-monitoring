import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { fetchPortfolioOverview } from '../services'
import type { PortfolioQueryParams } from '../types'

export const PORTFOLIO_OVERVIEW_QUERY_KEY = ['portfolio-overview']

export function usePortfolioOverview(params: PortfolioQueryParams) {
  return useQuery({
    queryKey: [...PORTFOLIO_OVERVIEW_QUERY_KEY, params],
    queryFn: () => fetchPortfolioOverview(params),
    placeholderData: keepPreviousData,
  })
}
