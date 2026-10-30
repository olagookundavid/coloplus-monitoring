import type { ComponentType, ReactNode, SVGProps } from 'react'

import { Button } from './Button'
import { ButtonLink } from './ButtonLink'

export type AlertBannerTone = 'success' | 'info' | 'warning' | 'danger'

export interface AlertBannerAction {
  label: string
  to?: string
  onClick?: () => void
}

export interface AlertBannerProps {
  tone: AlertBannerTone
  icon?: ComponentType<SVGProps<SVGSVGElement>>
  title: ReactNode
  description?: ReactNode
  action?: AlertBannerAction
  className?: string
}

const TONE_CLASSES: Record<AlertBannerTone, string> = {
  success: 'bg-primary-soft border-primary text-primary-700',
  info: 'bg-surface-subtle border-border text-text',
  warning: 'bg-warning-soft border-warning text-warning-700',
  danger: 'bg-danger-50 border-danger text-danger-600',
}

export function AlertBanner({
  tone,
  icon: Icon,
  title,
  description,
  action,
  className = '',
}: AlertBannerProps) {
  return (
    <div
      role={tone === 'danger' || tone === 'warning' ? 'alert' : 'status'}
      className={`flex flex-col gap-2 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between ${TONE_CLASSES[tone]} ${className}`}
    >
      <div className="flex items-start gap-2">
        {Icon && <Icon className="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />}
        <p className="text-sm">
          <span className="font-semibold">{title}</span>
          {description && <span className="ml-1 font-normal">{description}</span>}
        </p>
      </div>
      {action &&
        (action.to ? (
          <ButtonLink
            to={action.to}
            variant="link"
            size="sm"
            className="shrink-0 self-start sm:self-auto"
          >
            {action.label}
          </ButtonLink>
        ) : (
          <Button
            variant="link"
            size="sm"
            onClick={action.onClick}
            className="shrink-0 self-start sm:self-auto"
          >
            {action.label}
          </Button>
        ))}
    </div>
  )
}
