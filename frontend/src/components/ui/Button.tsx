import type { ButtonHTMLAttributes, ComponentType, SVGProps } from 'react'

import { Spinner } from './Spinner'

export type ButtonVariant =
  'primary' | 'secondary' | 'outline' | 'danger' | 'ghost' | 'link' | 'success' | 'info' | 'warning'
export type ButtonSize = 'sm' | 'md' | 'lg'
export type ButtonTone = 'light' | 'dark'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  icon?: ComponentType<SVGProps<SVGSVGElement>>
  iconPosition?: 'left' | 'right'
  loading?: boolean
  fullWidth?: boolean
  tone?: ButtonTone
}

// success/info have no sourced brand token yet (frontend/CLAUDE.md §7,
// known gap) — fall back to plain Tailwind palette rather than invent a
// semantic token. Neither is used by the auth screens.
const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-white hover:bg-primary-700 disabled:bg-primary/60',
  secondary:
    'bg-surface-subtle text-text border border-border hover:bg-border/40 disabled:opacity-60',
  outline: 'bg-surface text-text border border-border hover:bg-surface-subtle disabled:opacity-60',
  danger: 'bg-danger text-white hover:bg-danger-600/90 disabled:bg-danger/60',
  ghost: 'bg-transparent text-text hover:bg-surface-subtle disabled:opacity-60',
  link: 'bg-transparent text-primary hover:underline p-0! h-auto! disabled:opacity-60',
  success: 'bg-emerald-600 text-white hover:bg-emerald-700 disabled:bg-emerald-600/60',
  info: 'bg-sky-600 text-white hover:bg-sky-700 disabled:bg-sky-600/60',
  warning: 'bg-warning text-white hover:bg-warning-500/90 disabled:bg-warning/60',
}

const DARK_VARIANT_CLASSES: Partial<Record<ButtonVariant, string>> = {
  outline: 'bg-transparent text-white border border-white/40 hover:bg-white/10',
  ghost: 'bg-transparent text-white hover:bg-white/10',
  link: 'bg-transparent text-white hover:underline p-0! h-auto!',
}

const SIZE_CLASSES: Record<ButtonSize, string> = {
  sm: 'h-9 px-3 text-sm gap-1.5',
  md: 'h-11 px-4 text-sm gap-2',
  lg: 'h-12 px-6 text-base gap-2',
}

export function Button({
  variant = 'primary',
  size = 'md',
  icon: Icon,
  iconPosition = 'left',
  loading = false,
  fullWidth = false,
  tone = 'light',
  type = 'button',
  disabled,
  className = '',
  children,
  ...rest
}: ButtonProps) {
  const variantClass =
    tone === 'dark' && DARK_VARIANT_CLASSES[variant]
      ? DARK_VARIANT_CLASSES[variant]
      : VARIANT_CLASSES[variant]

  return (
    <button
      type={type}
      disabled={disabled || loading}
      className={`inline-flex cursor-pointer items-center justify-center rounded-lg font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary disabled:cursor-not-allowed ${SIZE_CLASSES[size]} ${variantClass} ${fullWidth ? 'w-full' : ''} ${className}`}
      {...rest}
    >
      {loading ? (
        <Spinner className="h-4 w-4" />
      ) : (
        Icon && iconPosition === 'left' && <Icon className="h-4 w-4" aria-hidden="true" />
      )}
      {children}
      {!loading && Icon && iconPosition === 'right' && (
        <Icon className="h-4 w-4" aria-hidden="true" />
      )}
    </button>
  )
}
