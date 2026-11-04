import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import type { PortfolioSite } from '../../types'
import { AlertBanner } from '../ui'

export interface SiteAttentionBannerProps {
  sites: PortfolioSite[]
}

// The count is computed from the real PortfolioSite[] list, not a hardcoded
// string -- two pages render this banner, and a literal number duplicated
// in both would be exactly the kind of figure this project's own trust
// rules exist to prevent (frontend/CLAUDE.md §11/§12). Renders nothing when
// no site currently needs attention.
export function SiteAttentionBanner({ sites }: SiteAttentionBannerProps) {
  const attentionCount = sites.filter((site) => site.condition === 'outage_reported').length

  if (attentionCount === 0) return null

  return (
    <AlertBanner
      tone="warning"
      icon={ExclamationTriangleIcon}
      title={`${attentionCount} site${attentionCount === 1 ? '' : 's'} need${attentionCount === 1 ? 's' : ''} attention.`}
      description="An outage is being addressed."
      action={{ label: 'View issues', to: ROUTES.CUSTOMER_SITES }}
    />
  )
}
