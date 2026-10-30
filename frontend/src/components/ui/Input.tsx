import type { InputHTMLAttributes } from 'react'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
  tone?: 'light' | 'dark'
}

export function Input({ invalid = false, tone = 'light', className = '', ...rest }: InputProps) {
  const toneClass =
    tone === 'dark'
      ? 'bg-white/10 text-white placeholder-white/60 border-white/30 focus:border-white'
      : 'bg-surface text-text placeholder-text-muted border-border focus:border-primary'
  return (
    <input
      className={`h-11 w-full rounded-lg border px-3 text-base transition-colors focus:outline-none sm:text-sm ${toneClass} ${invalid ? 'border-danger focus:border-danger' : ''} ${className}`}
      aria-invalid={invalid || undefined}
      {...rest}
    />
  )
}
