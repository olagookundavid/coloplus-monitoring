import type { ComponentType, KeyboardEvent, SVGProps } from 'react'

export interface TabItem<T extends string = string> {
  value: T
  label: string
  icon?: ComponentType<SVGProps<SVGSVGElement>>
}

export interface TabsProps<T extends string = string> {
  items: TabItem<T>[]
  value: T
  onChange: (value: T) => void
  'aria-label': string
  tone?: 'light' | 'dark'
}

export function Tabs<T extends string = string>({
  items,
  value,
  onChange,
  'aria-label': ariaLabel,
  tone = 'light',
}: TabsProps<T>) {
  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const index = items.findIndex((item) => item.value === value)
    if (index === -1) return

    let nextIndex: number | null = null
    if (event.key === 'ArrowRight') nextIndex = (index + 1) % items.length
    else if (event.key === 'ArrowLeft') nextIndex = (index - 1 + items.length) % items.length
    else if (event.key === 'Home') nextIndex = 0
    else if (event.key === 'End') nextIndex = items.length - 1

    if (nextIndex !== null) {
      event.preventDefault()
      const nextItem = items[nextIndex]
      if (nextItem) onChange(nextItem.value)
    }
  }

  const inactiveClass =
    tone === 'dark' ? 'text-white/70 hover:text-white' : 'text-text-muted hover:text-text'
  const activeClass = tone === 'dark' ? 'text-white border-white' : 'text-primary border-primary'
  const trackClass = tone === 'dark' ? 'border-white/20' : 'border-border'

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      onKeyDown={handleKeyDown}
      className={`inline-flex gap-4 border-b ${trackClass}`}
    >
      {items.map((item) => {
        const isActive = item.value === value
        const Icon = item.icon
        return (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            tabIndex={isActive ? 0 : -1}
            onClick={() => onChange(item.value)}
            className={`-mb-px flex cursor-pointer items-center gap-1.5 border-b-2 px-1 pb-2 text-sm font-medium transition-colors ${isActive ? activeClass : `border-transparent ${inactiveClass}`}`}
          >
            {Icon && <Icon className="h-4 w-4" aria-hidden="true" />}
            {item.label}
          </button>
        )
      })}
    </div>
  )
}
