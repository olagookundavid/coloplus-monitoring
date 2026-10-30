import type { InputHTMLAttributes, ReactNode } from 'react'

export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: ReactNode
  hint?: ReactNode
}

// The label and hint make one labelled row, and the whole row -- not just
// the small box -- is the tap target (frontend/CLAUDE.md §7).
export function Checkbox({ label, hint, id, className = '', ...rest }: CheckboxProps) {
  return (
    <label htmlFor={id} className={`flex min-h-11 cursor-pointer items-start gap-2.5 ${className}`}>
      <input
        id={id}
        type="checkbox"
        className="mt-0.5 h-4 w-4 shrink-0 rounded border-border accent-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        {...rest}
      />
      <span className="flex flex-col justify-center">
        <span className="text-sm font-medium text-text">{label}</span>
        {hint && <span className="text-sm text-text-muted">{hint}</span>}
      </span>
    </label>
  )
}
