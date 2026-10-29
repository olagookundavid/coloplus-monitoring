import type { ComponentType, SVGProps } from 'react'
import { Link, type LinkProps } from 'react-router-dom'

import type { ButtonSize, ButtonTone, ButtonVariant } from './Button'

export interface ButtonLinkProps extends LinkProps {
  variant?: ButtonVariant
  size?: ButtonSize
  icon?: ComponentType<SVGProps<SVGSVGElement>>
  iconPosition?: 'left' | 'right'
  fullWidth?: boolean
  tone?: ButtonTone
  className?: string
}

const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-white hover:bg-primary-700',
  secondary: 'bg-surface-subtle text-text border border-border hover:bg-border/40',
  outline: 'bg-surface text-text border border-border hover:bg-surface-subtle',
  danger: 'bg-danger text-white hover:bg-danger-600/90',
  ghost: 'bg-transparent text-text hover:bg-surface-subtle',
  link: 'bg-transparent text-primary hover:underline p-0! h-auto!',
  success: 'bg-emerald-600 text-white hover:bg-emerald-700',
  info: 'bg-sky-600 text-white hover:bg-sky-700',
  warning: 'bg-warning text-white hover:bg-warning-500/90',
}

const SIZE_CLASSES: Record<ButtonSize, string> = {
  sm: 'h-9 px-3 text-sm gap-1.5',
  md: 'h-11 px-4 text-sm gap-2',
  lg: 'h-12 px-6 text-base gap-2',
}

export function ButtonLink({
  variant = 'primary',
  size = 'md',
  icon: Icon,
  iconPosition = 'left',
  fullWidth = false,
  className = '',
  children,
  ...rest
}: ButtonLinkProps) {
  return (
    <Link
      className={`inline-flex cursor-pointer items-center justify-center rounded-lg font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary ${SIZE_CLASSES[size]} ${VARIANT_CLASSES[variant]} ${fullWidth ? 'w-full' : ''} ${className}`}
      {...rest}
    >
      {Icon && iconPosition === 'left' && <Icon className="h-4 w-4" aria-hidden="true" />}
      {children}
      {Icon && iconPosition === 'right' && <Icon className="h-4 w-4" aria-hidden="true" />}
    </Link>
  )
}
