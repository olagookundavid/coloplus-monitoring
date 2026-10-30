import Joi from 'joi'

// docs/DECISIONS.md 041 — Joi, chosen by the owner over a hand-rolled
// zero-dependency alternative.

export interface LoginFormValues {
  email: string
  password: string
}

export interface MfaCodeFormValues {
  code: string
}

export interface ForgotPasswordFormValues {
  email: string
}

export interface SetPasswordFormValues {
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

export const mfaCodeSchema = Joi.object<MfaCodeFormValues>({
  code: Joi.string().trim().required().messages({
    'string.empty': 'Enter the code from your authenticator app, or a recovery code.',
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

// Mirrors the backend's actual policy (12-72 chars, at least one letter and
// one digit) so a form error never slips through client-side only to be
// rejected again by the server.
export const setPasswordSchema = Joi.object<SetPasswordFormValues>({
  password: Joi.string()
    .min(12)
    .max(72)
    .pattern(/[A-Za-z]/, { name: 'letter' })
    .pattern(/[0-9]/, { name: 'digit' })
    .required()
    .messages({
      'string.empty': 'Enter a password.',
      'string.min': 'Password must be at least 12 characters.',
      'string.max': 'Password must be at most 72 characters.',
      'string.pattern.name': 'Password must include at least one letter and one number.',
    }),
  confirmPassword: Joi.any().valid(Joi.ref('password')).required().messages({
    'any.only': 'Passwords must match.',
    'any.required': 'Confirm your password.',
  }),
})
