/**
 * One source for "As of [time] WAT" everywhere it appears (the filter row,
 * the footer) so they can never disagree. A frozen mock instant, not
 * Date.now() — this data isn't live yet, so a ticking clock would
 * misrepresent freshness that isn't real (frontend/CLAUDE.md §11).
 */

const MOCK_SNAPSHOT_ISO = '2026-10-24T10:42:00+01:00'

export interface DataSnapshot {
  iso: string
  timeLabel: string
  dateLabel: string
  timezoneLabel: string
}

export function useDataSnapshot(): DataSnapshot {
  const date = new Date(MOCK_SNAPSHOT_ISO)
  return {
    iso: MOCK_SNAPSHOT_ISO,
    timeLabel: date.toLocaleTimeString('en-GB', {
      hour: '2-digit',
      minute: '2-digit',
      timeZone: 'Africa/Lagos',
    }),
    dateLabel: date.toLocaleDateString('en-GB', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      timeZone: 'Africa/Lagos',
    }),
    timezoneLabel: 'Africa/Lagos (WAT)',
  }
}
