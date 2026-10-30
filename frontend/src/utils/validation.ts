import type Joi from 'joi'

// Generic over any Joi schema — shared by auth and ticket forms (and any
// future one), not owned by a single domain's validation file.
export function getFieldErrors<T extends object>(
  schema: Joi.ObjectSchema<T>,
  values: T
): Record<string, string> {
  const { error } = schema.validate(values, { abortEarly: false })
  if (!error) return {}
  const fieldErrors: Record<string, string> = {}
  for (const detail of error.details) {
    const key = detail.path.join('.')
    if (!fieldErrors[key]) fieldErrors[key] = detail.message
  }
  return fieldErrors
}
