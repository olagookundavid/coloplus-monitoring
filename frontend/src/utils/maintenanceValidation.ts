import Joi from 'joi'

import type { MaintenanceRules, MaintenanceWindow } from '../types'

export interface RequestMaintenanceFormValues {
  siteId: string
  startsAt: string
  endsAt: string
  reason: string
}

export const requestMaintenanceSchema = Joi.object<RequestMaintenanceFormValues>({
  siteId: Joi.string().required().messages({
    'string.empty': 'Choose a site.',
  }),
  startsAt: Joi.date().iso().required().messages({
    'date.base': 'Say when the window starts.',
    'any.required': 'Say when the window starts.',
  }),
  endsAt: Joi.date().iso().required().greater(Joi.ref('startsAt')).messages({
    'date.base': 'Say when the window ends.',
    'date.greater': 'The end time must be after the start time.',
    'any.required': 'Say when the window ends.',
  }),
  reason: Joi.string().trim().min(5).required().messages({
    'string.empty': 'Say what the maintenance is.',
    'string.min': 'Say what the maintenance is (at least 5 characters).',
  }),
})

// A courtesy check against the list response's own `rules` and the site's
// other windows -- the API is still the authority and its 400 names the
// broken rule; this only saves a round trip for the obvious cases.
// Excludes rejected/cancelled windows, which don't count against the cap or
// an overlap. Silent (no errors) until both startsAt and endsAt parse and
// rules are known.
export function maintenanceRuleErrors(
  values: RequestMaintenanceFormValues,
  rules: MaintenanceRules | undefined,
  existingWindows: MaintenanceWindow[]
): Partial<Record<'startsAt' | 'endsAt', string>> {
  const errors: Partial<Record<'startsAt' | 'endsAt', string>> = {}
  if (!rules) return errors

  const start = new Date(values.startsAt)
  const end = new Date(values.endsAt)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end <= start) return errors

  const noticeHoursGiven = (start.getTime() - Date.now()) / 3_600_000
  if (noticeHoursGiven < rules.noticeHours) {
    errors.startsAt = `Needs at least ${rules.noticeHours}h notice.`
  }

  const durationHours = (end.getTime() - start.getTime()) / 3_600_000
  if (durationHours > rules.maxHours) {
    errors.endsAt = `A window is at most ${rules.maxHours}h.`
  } else if (
    start.getFullYear() !== end.getFullYear() ||
    start.getMonth() !== end.getMonth() ||
    start.getDate() !== end.getDate()
  ) {
    errors.endsAt = 'A window must stay within one day.'
  }

  const active = existingWindows.filter((w) => w.status === 'requested' || w.status === 'approved')

  const overlaps = active.some((w) => {
    const wStart = new Date(w.startsAt).getTime()
    const wEnd = new Date(w.endsAt).getTime()
    return start.getTime() < wEnd && end.getTime() > wStart
  })
  if (overlaps) {
    errors.startsAt = errors.startsAt ?? 'Overlaps another window at this site.'
  }

  const monthStart = new Date(start.getFullYear(), start.getMonth(), 1)
  const monthEnd = new Date(start.getFullYear(), start.getMonth() + 1, 0, 23, 59, 59)
  const monthlyHoursSoFar = active
    .filter((w) => {
      const wStart = new Date(w.startsAt)
      return wStart >= monthStart && wStart <= monthEnd
    })
    .reduce((sum, w) => sum + w.minutes / 60, 0)
  if (!errors.endsAt && monthlyHoursSoFar + durationHours > rules.monthlyCapHours) {
    errors.endsAt = `Exceeds the ${rules.monthlyCapHours}h/month cap for this site.`
  }

  return errors
}
