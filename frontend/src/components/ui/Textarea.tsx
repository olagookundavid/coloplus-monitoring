import type { TextareaHTMLAttributes } from 'react'

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  invalid?: boolean
  tone?: 'light' | 'dark'
}

export function Textarea({
  invalid = false,
  tone = 'light',
  rows = 4,
  className = '',
  ...rest
}: TextareaProps) {
  const toneClass =
    tone === 'dark'
      ? 'bg-white/10 text-white placeholder-white/60 border-white/30 focus:border-white'
      : 'bg-surface text-text placeholder-text-muted border-border focus:border-primary'
  return (
    <textarea
      rows={rows}
      className={`w-full rounded-lg border px-3 py-2 text-base transition-colors focus:outline-none sm:text-sm ${toneClass} ${invalid ? 'border-danger focus:border-danger' : ''} ${className}`}
      aria-invalid={invalid || undefined}
      {...rest}
    />
  )
}
