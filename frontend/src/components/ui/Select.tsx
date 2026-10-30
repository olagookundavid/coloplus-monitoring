import type { ComponentType, SelectHTMLAttributes, SVGProps } from 'react'
import { ChevronDownIcon } from '@heroicons/react/24/outline'

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  invalid?: boolean
  tone?: 'light' | 'dark'
  icon?: ComponentType<SVGProps<SVGSVGElement>>
}

export function Select({
  invalid = false,
  tone = 'light',
  icon: Icon,
  className = '',
  children,
  ...rest
}: SelectProps) {
  const toneClass =
    tone === 'dark'
      ? 'bg-white/10 text-white border-white/30 focus:border-white'
      : 'bg-surface text-text border-border focus:border-primary'
  const iconColorClass = tone === 'dark' ? 'text-white/70' : 'text-text-muted'

  return (
    // The leading icon lives inside this same positioning context as the
    // chevron (not as an external sibling) — a Select rendered with an
    // external prefix icon paints this wrapper div on top of it, since both
    // are positioned elements and this one comes later in the DOM.
    <div className="relative">
      {Icon && (
        <Icon
          className={`pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 ${iconColorClass}`}
          aria-hidden="true"
        />
      )}
      <select
        className={`h-11 w-full appearance-none rounded-lg border px-3 ${Icon ? 'pl-9' : ''} pr-9 text-base transition-colors focus:outline-none sm:text-sm ${toneClass} ${invalid ? 'border-danger focus:border-danger' : ''} ${className}`}
        aria-invalid={invalid || undefined}
        {...rest}
      >
        {children}
      </select>
      <ChevronDownIcon
        className={`pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 ${iconColorClass}`}
        aria-hidden="true"
      />
    </div>
  )
}
