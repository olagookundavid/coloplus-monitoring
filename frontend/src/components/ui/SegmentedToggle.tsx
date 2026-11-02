export interface SegmentedToggleOption<T extends string = string> {
  value: T
  label: string
}

export interface SegmentedToggleProps<T extends string = string> {
  options: [SegmentedToggleOption<T>, SegmentedToggleOption<T>]
  value: T | null
  onChange: (value: T) => void
  'aria-label': string
  disabled?: boolean
}

// A two-way switch for a single either/or choice (e.g. "Power on all day" /
// "Had an outage") -- the one control shape between a Checkbox and Tabs that
// this project doesn't otherwise have a primitive for.
export function SegmentedToggle<T extends string = string>({
  options,
  value,
  onChange,
  'aria-label': ariaLabel,
  disabled = false,
}: SegmentedToggleProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      aria-disabled={disabled || undefined}
      className="inline-flex w-full flex-col gap-2 rounded-lg border border-border bg-surface-subtle p-1 sm:flex-row"
    >
      {options.map((option) => {
        const selected = option.value === value
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            disabled={disabled}
            onClick={() => onChange(option.value)}
            className={`min-h-11 flex-1 rounded-md px-3 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
              disabled ? '' : 'cursor-pointer'
            } ${
              selected
                ? 'bg-primary-soft text-primary-700'
                : 'bg-transparent text-text-muted hover:text-text'
            }`}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
