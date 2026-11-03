import Joi from 'joi'

import type { PowerSource, SiteStatus, UnreportedTimePolicy } from '../types'

// Mirrors domain.Site.Validate() (backend/internal/domain/site.go) so
// errors surface before a network round trip.
export interface SiteProfileFormValues {
  airtelCode: string
  coloplusCode: string
  name: string
  address: string
  region: string
  state: string
  siteType: string
  powerSources: PowerSource[]
  responsibleTeam: string
  unreportedTime: UnreportedTimePolicy
  /** Raw input; '' means "not set." */
  consumptionLPH: string
  tankCapacityLitres: string
  /** When true, serviceStartTime/serviceEndTime are ignored and the window
      submits as the full 0-1440 default -- the backend's own default for a
      site with no stated window. */
  allDay: boolean
  serviceStartTime: string
  serviceEndTime: string
  freshnessHours: string
  status: SiteStatus
  commissioningDate: string
  latitude: string
  longitude: string
  generatorCapacity: string
  accessContactName: string
  accessContactPhone: string
  supervisorName: string
}

// Joi.any() here, not Joi.string(): these are optional numeric-as-text
// fields, and Joi.string() rejects '' outright (even with .allow('')
// present, which only whitelists the value and would skip .custom()
// entirely rather than letting it run) -- Joi.any() hands every value,
// including '', to the custom function below so the "not set" case and
// the range check live in one place.
function optionalPositiveNumber(max: number, message: string) {
  return Joi.any().custom((value: string, helpers) => {
    if (value === '') return value
    const n = Number(value)
    if (Number.isNaN(n) || n <= 0 || n > max) return helpers.error('any.invalid')
    return value
  }, 'optional positive number').messages({ 'any.invalid': message })
}

export const siteProfileSchema = Joi.object<SiteProfileFormValues>({
  airtelCode: Joi.string().trim().required().messages({
    'string.empty': 'Enter the Airtel code.',
  }),
  coloplusCode: Joi.string().trim().required().messages({
    'string.empty': 'Enter the Coloplus code.',
  }),
  name: Joi.string().trim().required().messages({
    'string.empty': 'Enter the site name.',
  }),
  address: Joi.string().allow('').trim(),
  region: Joi.string().allow('').trim(),
  state: Joi.string().trim().required().messages({
    'string.empty': 'Enter the state.',
  }),
  siteType: Joi.string().allow('').trim(),
  powerSources: Joi.array().items(Joi.string()).min(1).messages({
    'array.min': 'Select at least one power source.',
  }),
  responsibleTeam: Joi.string().allow('').trim(),
  unreportedTime: Joi.string().valid('available', 'unknown').required(),
  consumptionLPH: optionalPositiveNumber(1000, 'Must be more than 0 and at most 1,000 L/h.'),
  tankCapacityLitres: optionalPositiveNumber(
    1_000_000,
    'Must be more than 0 and at most 1,000,000 L.'
  ),
  allDay: Joi.boolean().required(),
  serviceStartTime: Joi.any().custom((value: string, helpers) => {
    const { allDay } = helpers.state.ancestors[0] as SiteProfileFormValues
    if (!allDay && !value) return helpers.error('any.invalid')
    return value
  }, 'required unless all day').messages({ 'any.invalid': 'Set a start time.' }),
  serviceEndTime: Joi.any().custom((value: string, helpers) => {
    const { allDay, serviceStartTime } = helpers.state.ancestors[0] as SiteProfileFormValues
    if (allDay) return value
    if (!value) return helpers.error('any.required')
    if (serviceStartTime && value <= serviceStartTime) return helpers.error('any.invalid')
    return value
  }, 'required unless all day, after start').messages({
    'any.required': 'Set an end time.',
    'any.invalid': 'The end time must be after the start time.',
  }),
  freshnessHours: Joi.any().custom((value: string, helpers) => {
    const n = Number(value)
    if (!value || Number.isNaN(n) || n < 1) return helpers.error('any.invalid')
    return value
  }, 'minimum 1').messages({ 'any.invalid': 'Must be at least 1 hour.' }),
  status: Joi.string().valid('pending_acceptance', 'active', 'suspended').required(),
  commissioningDate: Joi.string().allow(''),
  latitude: Joi.any().custom((value: string, helpers) => {
    const { longitude } = helpers.state.ancestors[0] as SiteProfileFormValues
    if (value === '' && longitude === '') return value
    if (value === '') return helpers.error('any.pair')
    const n = Number(value)
    if (Number.isNaN(n) || n < -90 || n > 90) return helpers.error('any.invalid')
    return value
  }, 'latitude').messages({
    'any.invalid': 'Latitude must be between -90 and 90.',
    'any.pair': 'Set both latitude and longitude, or neither.',
  }),
  longitude: Joi.any().custom((value: string, helpers) => {
    const { latitude } = helpers.state.ancestors[0] as SiteProfileFormValues
    if (value === '' && latitude === '') return value
    if (value === '') return helpers.error('any.pair')
    const n = Number(value)
    if (Number.isNaN(n) || n < -180 || n > 180) return helpers.error('any.invalid')
    return value
  }, 'longitude').messages({
    'any.invalid': 'Longitude must be between -180 and 180.',
    'any.pair': 'Set both latitude and longitude, or neither.',
  }),
  generatorCapacity: Joi.string().allow('').trim(),
  accessContactName: Joi.string().allow('').trim(),
  accessContactPhone: Joi.string().allow('').trim(),
  supervisorName: Joi.string().allow('').trim(),
})
