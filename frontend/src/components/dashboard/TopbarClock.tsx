import { useEffect, useState } from 'react'

// Africa/Lagos explicitly, not the browser's local zone — this app is
// WAT-anchored everywhere else (frontend/CLAUDE.md §5's "as of ... WAT"
// rule), so a live clock in any other zone would silently mislead anyone
// viewing it from outside WAT.
const TIME_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Africa/Lagos',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: true,
})

const DATE_FORMATTER = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Africa/Lagos',
  weekday: 'short',
  day: '2-digit',
  month: 'short',
  year: 'numeric',
})

export function TopbarClock() {
  const [now, setNow] = useState(() => new Date())

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])

  return (
    <div className="hidden flex-col leading-tight md:flex">
      <span className="text-sm font-semibold text-text">
        {TIME_FORMATTER.format(now).toLowerCase()}
      </span>
      <span className="text-xs text-text-muted">{DATE_FORMATTER.format(now)}</span>
    </div>
  )
}
