// A minutes count from a PeriodSummary (assumed time, planned maintenance,
// source hours, ...) shown as "3h 20m" / "45m" rather than raw minutes.
export function formatMinutes(minutes: number): string {
  if (minutes <= 0) return '0m'
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (hours === 0) return `${mins}m`
  if (mins === 0) return `${hours}h`
  return `${hours}h ${mins}m`
}
