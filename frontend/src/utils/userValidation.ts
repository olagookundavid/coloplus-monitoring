import Joi from 'joi'

import { STAFF_ROLES, type StaffRole } from '../types'

export interface InviteUserFormValues {
  email: string
  name: string
  role: StaffRole
}

export const inviteUserSchema = Joi.object<InviteUserFormValues>({
  email: Joi.string()
    .trim()
    .email({ tlds: { allow: false } })
    .required()
    .messages({
      'string.empty': 'Enter an email address.',
      'string.email': 'Enter a valid email address.',
    }),
  name: Joi.string().trim().required().messages({
    'string.empty': 'Enter their full name.',
  }),
  role: Joi.string()
    .valid(...STAFF_ROLES)
    .required()
    .messages({
      'any.only': 'Choose a role.',
      'string.empty': 'Choose a role.',
    }),
})
