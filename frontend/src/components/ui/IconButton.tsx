import type { ButtonHTMLAttributes, ComponentType, SVGProps } from 'react'

export type IconButtonVariant = 'ghost' | 'secondary' | 'danger'

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  label: string
  variant?: IconButtonVariant
}

const VARIANT_CLASSES: Record<IconButtonVariant, string> = {
  ghost: 'bg-transparent text-text-muted hover:bg-surface-subtle hover:text-text',
  secondary: 'bg-surface-subtle text-text hover:bg-border/40',
  danger: 'bg-transparent text-danger hover:bg-danger-50',
}

export function IconButton({
  icon: Icon,
  label,
  variant = 'ghost',
  type = 'button',
  className = '',
  ...rest
}: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      className={`inline-flex h-11 w-11 min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-lg transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary disabled:cursor-not-allowed disabled:opacity-60 ${VARIANT_CLASSES[variant]} ${className}`}
      {...rest}
    >
      <Icon className="h-5 w-5" aria-hidden="true" />
    </button>
  )
}
