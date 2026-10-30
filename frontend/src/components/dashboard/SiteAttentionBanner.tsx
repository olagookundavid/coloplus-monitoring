import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { ROUTES } from '../../constants'
import type { Site } from '../../types'
import { AlertBanner } from '../ui'

export interface SiteAttentionBannerProps {
  sites: Site[]
}

const ATTENTION_STATUSES: Site['status'][] = ['Power outage', 'Low diesel']

// The count is computed from the real Site[] list, not a hardcoded string —
// two pages render this banner, and a literal "2" duplicated in both would
// be exactly the kind of number this project's own trust rules exist to
// prevent (frontend/CLAUDE.md §11/§12). Renders nothing when no site
// currently needs attention.
export function SiteAttentionBanner({ sites }: SiteAttentionBannerProps) {
  const attentionCount = sites.filter((site) => ATTENTION_STATUSES.includes(site.status)).length

  if (attentionCount === 0) return null

  return (
    <AlertBanner
      tone="warning"
      icon={ExclamationTriangleIcon}
      title={`${attentionCount} site${attentionCount === 1 ? '' : 's'} need${attentionCount === 1 ? 's' : ''} attention.`}
      description="An active power fault is being addressed."
      action={{ label: 'View issues', to: ROUTES.CUSTOMER_SITES }}
    />
  )
}
