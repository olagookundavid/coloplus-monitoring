export interface User {
  id: string
  email: string
  name: string
  role: string
  isActive: boolean
  mfaEnabled: boolean
  invited: boolean
  lastLoginAt?: string
}

export interface LoginRequest {
  email: string
  password: string
  totpCode?: string
}

// Discriminated on `status`: a password-correct-but-code-needed attempt
// carries no session at all (no cookie is set) — the caller resubmits the
// same login call with `totpCode` filled in rather than hitting a separate
// "verify" endpoint.
export type LoginResult =
  | { status: 'mfa_required' }
  | {
      status: 'ok'
      mfaSetupRequired: boolean
      user: User
      accessToken: string
      expiresAt: string
    }

export interface ForgotPasswordRequest {
  email: string
}

// Shared by both the invite-acceptance and password-reset links — the
// backend tells them apart by the token's purpose, not by request shape.
export interface SetPasswordRequest {
  token: string
  password: string
}

export interface MeResult {
  user: User
  permissions: string[]
  allSites: boolean
  siteIds: string[]
  mfaSetupRequired: boolean
  recoveryCodesLeft: number
}

export interface MfaSetupResult {
  secret: string
  otpauthUrl: string
}

export interface EnableMfaRequest {
  code: string
}

// `recoveryCodes` is null when MFA was already enabled (the backend
// returns 200 with a null list rather than an error in that case).
export interface RecoveryCodesResult {
  recoveryCodes: string[] | null
}

export interface AuthApiErrorShape {
  message: string
  status: number
  requestId?: string
  fieldErrors?: Record<string, string>
}
