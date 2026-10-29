import Joi from 'joi'

// docs/DECISIONS.md 026 — Joi, chosen by the owner over a hand-rolled
// zero-dependency alternative.

export interface LoginFormValues {
  email: string
  password: string
}

export interface ForgotPasswordFormValues {
  email: string
}

export interface AcceptInviteFormValues {
  password: string
  confirmPassword: string
}

export const loginSchema = Joi.object<LoginFormValues>({
  email: Joi.string()
    .trim()
    .email({ tlds: { allow: false } })
    .required()
    .messages({
      'string.empty': 'Enter your email address.',
      'string.email': 'Enter a valid email address.',
    }),
  password: Joi.string().required().messages({
    'string.empty': 'Enter your password.',
  }),
})

export const forgotPasswordSchema = Joi.object<ForgotPasswordFormValues>({
  email: Joi.string()
    .trim()
    .email({ tlds: { allow: false } })
    .required()
    .messages({
      'string.empty': 'Enter your email address.',
      'string.email': 'Enter a valid email address.',
    }),
})

export const acceptInviteSchema = Joi.object<AcceptInviteFormValues>({
  password: Joi.string().min(8).required().messages({
    'string.empty': 'Enter a password.',
    'string.min': 'Password must be at least 8 characters.',
  }),
  confirmPassword: Joi.any().valid(Joi.ref('password')).required().messages({
    'any.only': 'Passwords must match.',
    'any.required': 'Confirm your password.',
  }),
})

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
