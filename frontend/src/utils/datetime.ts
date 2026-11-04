export function formatDateTimeWAT(iso: string): string {
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Africa/Lagos',
  })
}

// How long ago an ISO timestamp was, in whole hours under a day and whole
// days beyond that -- used to flag approvals waiting too long.
export function formatAge(iso?: string): string {
  if (!iso) return '—'
  const ms = Date.now() - new Date(iso).getTime()
  const hours = Math.max(0, Math.floor(ms / 3_600_000))
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

// A stacked time/day cell (e.g. "09:04" / "Today") for audit-style tables --
// WAT, same convention as formatDateTimeWAT.
export function formatAuditWhen(iso: string): { time: string; day: string } {
  const date = new Date(iso)
  const time = date.toLocaleString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Africa/Lagos',
  })

  const dayKey = (d: Date) => d.toLocaleDateString('en-CA', { timeZone: 'Africa/Lagos' })
  const today = new Date()
  const yesterday = new Date(today.getTime() - 24 * 3_600_000)

  let day: string
  if (dayKey(date) === dayKey(today)) {
    day = 'Today'
  } else if (dayKey(date) === dayKey(yesterday)) {
    day = 'Yesterday'
  } else {
    day = date.toLocaleDateString('en-GB', {
      weekday: 'short',
      day: 'numeric',
      month: 'short',
      timeZone: 'Africa/Lagos',
    })
  }

  return { time, day }
}

// Snaps any "YYYY-MM-DD" to that week's Monday -- pure UTC calendar math,
// deliberately not going through a local-timezone-sensitive Date method,
// since this is calendar arithmetic, not a display timestamp. Used so a
// weekly export's week_start (which the backend requires to be a Monday)
// can never fail that check just because of which day was picked.
export function mondayOf(dateStr: string): string {
  const parts = dateStr.split('-')
  const year = Number(parts[0])
  const month = Number(parts[1])
  const day = Number(parts[2])
  const date = new Date(Date.UTC(year, month - 1, day))
  const weekday = date.getUTCDay() // 0 = Sunday .. 6 = Saturday
  const daysSinceMonday = weekday === 0 ? 6 : weekday - 1
  date.setUTCDate(date.getUTCDate() - daysSinceMonday)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`
}

// Finer-grained than formatAge -- minutes under an hour, hours under a day,
// "Yesterday", then whole days -- for compact feeds like the notifications
// dropdown.
export function formatRelativeShort(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const minutes = Math.floor(ms / 60_000)
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} h ago`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'Yesterday'
  return `${days} d ago`
}
