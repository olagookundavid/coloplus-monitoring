import type { SelectHTMLAttributes } from 'react'
import { ChevronDownIcon } from '@heroicons/react/24/outline'

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  invalid?: boolean
  tone?: 'light' | 'dark'
}

export function Select({
  invalid = false,
  tone = 'light',
  className = '',
  children,
  ...rest
}: SelectProps) {
  const toneClass =
    tone === 'dark'
      ? 'bg-white/10 text-white border-white/30 focus:border-white'
      : 'bg-surface text-text border-border focus:border-primary'

  return (
    <div className="relative">
      <select
        className={`h-11 w-full appearance-none rounded-lg border px-3 pr-9 text-base transition-colors focus:outline-none sm:text-sm ${toneClass} ${invalid ? 'border-danger focus:border-danger' : ''} ${className}`}
        aria-invalid={invalid || undefined}
        {...rest}
      >
        {children}
      </select>
      <ChevronDownIcon
        className={`pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 ${tone === 'dark' ? 'text-white/70' : 'text-text-muted'}`}
        aria-hidden="true"
      />
    </div>
  )
}
