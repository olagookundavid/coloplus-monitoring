import Joi from 'joi'

import type { FuelEntryKind } from '../types'

export interface RecordFuelFormValues {
  kind: FuelEntryKind
  litres: string
  occurredAt: string
  note: string
}

export const recordFuelSchema = Joi.object<RecordFuelFormValues>({
  kind: Joi.string().valid('delivery', 'reading').required().messages({
    'string.empty': 'Select delivery or reading.',
    'any.only': 'Select delivery or reading.',
  }),
  litres: Joi.number()
    .greater(0)
    .required()
    .messages({
      'number.base': 'Enter the litres.',
      'number.greater': 'Litres must be more than 0.',
      'any.required': 'Enter the litres.',
    }),
  occurredAt: Joi.date().iso().max('now').required().messages({
    'date.base': 'Enter when this happened.',
    'date.max': "This can't be in the future.",
    'any.required': 'Enter when this happened.',
  }),
  note: Joi.string().trim().allow(''),
})

export const VOID_REASON_MIN_LENGTH = 5

export function voidReasonError(reason: string): string | undefined {
  if (reason.trim().length < VOID_REASON_MIN_LENGTH) {
    return `Say why this entry is being voided (at least ${VOID_REASON_MIN_LENGTH} characters).`
  }
  return undefined
}
