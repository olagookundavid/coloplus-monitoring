import { useRef } from 'react'
import type { ClipboardEvent, KeyboardEvent } from 'react'

export interface OtpInputProps {
  length?: number
  value: string
  onChange: (code: string) => void
  onComplete?: (code: string) => void
  invalid?: boolean
  disabled?: boolean
  id?: string
  autoFocus?: boolean
}

// A row of single-character boxes behind one value/onChange string API --
// callers never deal with per-box state. Every box stays focusable
// (disabling "future" boxes was tried and reverted: a disabled input
// can't receive focus(), and that call runs immediately after the state
// update, before the box's own disabled attribute has re-rendered, so
// auto-advance to the next box silently failed).
export function OtpInput({
  length = 6,
  value,
  onChange,
  onComplete,
  invalid = false,
  disabled = false,
  id,
  autoFocus = false,
}: OtpInputProps) {
  const boxRefs = useRef<Array<HTMLInputElement | null>>([])

  function setValue(next: string) {
    onChange(next)
    if (next.length === length) onComplete?.(next)
  }

  function handleChange(index: number, raw: string) {
    const char = raw.slice(-1)
    if (!char) return
    setValue(value.slice(0, index) + char)
    if (index < length - 1) boxRefs.current[index + 1]?.focus()
  }

  function handleKeyDown(index: number, event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Backspace') {
      if (value[index]) {
        event.preventDefault()
        setValue(value.slice(0, index) + value.slice(index + 1))
        boxRefs.current[index]?.focus()
      } else if (index > 0) {
        event.preventDefault()
        setValue(value.slice(0, index - 1))
        boxRefs.current[index - 1]?.focus()
      }
    } else if (event.key === 'ArrowLeft' && index > 0) {
      boxRefs.current[index - 1]?.focus()
    } else if (event.key === 'ArrowRight' && index < length - 1) {
      boxRefs.current[index + 1]?.focus()
    }
  }

  function handlePaste(event: ClipboardEvent<HTMLInputElement>) {
    event.preventDefault()
    const pasted = event.clipboardData.getData('text').replace(/\s/g, '').slice(0, length)
    if (!pasted) return
    setValue(pasted)
    boxRefs.current[Math.min(pasted.length, length - 1)]?.focus()
  }

  return (
    <div className="flex gap-2">
      {Array.from({ length }, (_, index) => (
        <input
          key={index}
          ref={(el) => {
            boxRefs.current[index] = el
          }}
          id={index === 0 ? id : undefined}
          aria-label={`Digit ${index + 1} of ${length}`}
          type="text"
          inputMode="numeric"
          autoComplete={index === 0 ? 'one-time-code' : 'off'}
          maxLength={1}
          value={value[index] ?? ''}
          disabled={disabled}
          autoFocus={autoFocus && index === 0}
          aria-invalid={invalid || undefined}
          onChange={(event) => handleChange(index, event.target.value)}
          onKeyDown={(event) => handleKeyDown(index, event)}
          onPaste={handlePaste}
          className={`h-12 w-11 rounded-lg border text-center text-lg font-semibold text-text transition-colors focus:outline-none disabled:cursor-not-allowed disabled:opacity-60 sm:text-base ${
            invalid
              ? 'border-danger focus:border-danger'
              : 'border-border bg-surface focus:border-primary'
          }`}
        />
      ))}
    </div>
  )
}
