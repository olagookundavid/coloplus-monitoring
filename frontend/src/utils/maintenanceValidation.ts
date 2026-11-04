import Joi from 'joi'

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
