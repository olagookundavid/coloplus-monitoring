import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { fetchPortfolioTrend } from '../services'
import type { PortfolioTrendQueryParams } from '../types'

export const PORTFOLIO_TREND_QUERY_KEY = ['portfolio-trend']

export function usePortfolioTrend(params: PortfolioTrendQueryParams) {
  return useQuery({
    queryKey: [...PORTFOLIO_TREND_QUERY_KEY, params],
    queryFn: () => fetchPortfolioTrend(params),
    placeholderData: keepPreviousData,
  })
}
