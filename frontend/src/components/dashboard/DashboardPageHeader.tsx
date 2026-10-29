import type { ReactNode } from 'react'
import { ChevronLeftIcon } from '@heroicons/react/24/outline'

import { ButtonLink } from '../ui'

export interface DashboardPageHeaderProps {
  eyebrow: string
  title: string
  subtitle?: ReactNode
  actions?: ReactNode
  back?: { to: string; label: string }
}

// frontend/CLAUDE.md §5 — exactly this prop list, no filter props bolted on
// (DashboardFilterBar is a sibling the page renders separately below).
export function DashboardPageHeader({
  eyebrow,
  title,
  subtitle,
  actions,
  back,
}: DashboardPageHeaderProps) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        {back && (
          <ButtonLink to={back.to} variant="link" size="sm" icon={ChevronLeftIcon} className="mb-2">
            {back.label}
          </ButtonLink>
        )}
        <p className="text-sm font-semibold uppercase tracking-wide text-primary-600">{eyebrow}</p>
        <h1 className="mt-1 text-2xl font-bold text-text sm:text-3xl">{title}</h1>
        {subtitle && <div className="mt-1 text-sm text-text-muted">{subtitle}</div>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2 sm:shrink-0">{actions}</div>}
    </div>
  )
}
