import Joi from 'joi'

import type { TicketCategory, TicketPriority } from '../types'

export interface CreateTicketFormValues {
  siteId: string
  category: TicketCategory
  subject: string
  priority: TicketPriority
  contactEmail: string
  description: string
}

export const createTicketSchema = Joi.object<CreateTicketFormValues>({
  siteId: Joi.string().required().messages({
    'string.empty': 'Select the affected site.',
  }),
  category: Joi.string().required().messages({
    'string.empty': 'Select a category.',
  }),
  subject: Joi.string().trim().required().messages({
    'string.empty': 'Briefly describe the issue.',
  }),
  priority: Joi.string().required().messages({
    'string.empty': 'Select a priority.',
  }),
  contactEmail: Joi.string()
    .trim()
    .email({ tlds: { allow: false } })
    .allow('')
    .messages({
      'string.email': 'Enter a valid email address.',
    }),
  description: Joi.string().trim().required().messages({
    'string.empty': 'Tell us what happened.',
  }),
})
