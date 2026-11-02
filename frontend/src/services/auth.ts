import { AxiosError } from 'axios'

import type {
  EnableMfaRequest,
  ForgotPasswordRequest,
  LoginRequest,
  LoginResult,
  MeResult,
  MfaSetupResult,
  RecoveryCodesResult,
  SetPasswordRequest,
  User,
  AuthApiErrorShape,
} from '../types'
import { getApi } from './api'

export class AuthApiError extends Error implements AuthApiErrorShape {
  status: number
  requestId?: string
  fieldErrors?: Record<string, string>

  constructor(shape: AuthApiErrorShape) {
    super(shape.message)
    this.name = 'AuthApiError'
    this.status = shape.status
    this.requestId = shape.requestId
    this.fieldErrors = shape.fieldErrors
  }
}

// The backend's generic envelope is {error, request_id}, but the one
// pre-auth case of a missing/invalid token, and every rate-limit 429,
// come from a different helper that only sends {error} — request_id is
// genuinely optional, not a bug to work around.
export function mapAxiosError(error: unknown, fallback: string): AuthApiError {
  if (error instanceof AxiosError) {
    if (!error.response) {
      return new AuthApiError({
        message: 'Could not reach the server. Check your connection and try again.',
        status: 0,
      })
    }
    const data = error.response.data as { error?: string; request_id?: string } | undefined
    return new AuthApiError({
      message: data?.error || fallback,
      status: error.response.status,
      requestId: data?.request_id,
    })
  }
  return new AuthApiError({ message: fallback, status: 0 })
}

interface UserDTO {
  id: string
  email: string
  full_name: string
  role: string
  is_active: boolean
  mfa_enabled: boolean
  invited: boolean
  last_login_at?: string
}

function mapUser(dto: UserDTO): User {
  return {
    id: dto.id,
    email: dto.email,
    name: dto.full_name,
    role: dto.role,
    isActive: dto.is_active,
    mfaEnabled: dto.mfa_enabled,
    invited: dto.invited,
    lastLoginAt: dto.last_login_at,
  }
}

export async function login(request: LoginRequest): Promise<LoginResult> {
  try {
    const { data } = await getApi().post('/auth/login', {
      email: request.email,
      password: request.password,
      totp_code: request.totpCode,
    })
    if (data.status === 'mfa_required') {
      return { status: 'mfa_required' }
    }
    return {
      status: 'ok',
      mfaSetupRequired: data.mfa_setup_required,
      user: mapUser(data.user),
      accessToken: data.access_token,
      expiresAt: data.expires_at,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Incorrect email or password.')
  }
}

export async function logout(): Promise<void> {
  try {
    await getApi().post('/auth/logout', {})
  } catch (error) {
    throw mapAxiosError(error, 'Something went wrong signing out.')
  }
}

export async function requestPasswordReset(request: ForgotPasswordRequest): Promise<void> {
  try {
    await getApi().post('/auth/forgot-password', request)
  } catch (error) {
    throw mapAxiosError(error, 'Something went wrong sending the reset link. Try again shortly.')
  }
}

export async function setPassword(request: SetPasswordRequest): Promise<void> {
  try {
    await getApi().post('/auth/set-password', request)
  } catch (error) {
    throw mapAxiosError(error, 'This link is invalid or has expired.')
  }
}

export async function fetchMe(): Promise<MeResult> {
  try {
    const { data } = await getApi().get('/auth/me')
    return {
      user: mapUser(data.user),
      permissions: data.permissions,
      allSites: data.all_sites,
      siteIds: data.site_ids,
      mfaSetupRequired: data.mfa_setup_required,
      recoveryCodesLeft: data.recovery_codes_left,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Your session has expired. Sign in again.')
  }
}

export async function setupMfa(): Promise<MfaSetupResult> {
  try {
    const { data } = await getApi().post('/auth/mfa/setup', {})
    return { secret: data.secret, otpauthUrl: data.otpauth_url }
  } catch (error) {
    throw mapAxiosError(error, 'Something went wrong starting two-step verification.')
  }
}

export async function enableMfa(request: EnableMfaRequest): Promise<RecoveryCodesResult> {
  try {
    const { data } = await getApi().post('/auth/mfa/enable', { code: request.code })
    return { recoveryCodes: data.recovery_codes ?? null }
  } catch (error) {
    throw mapAxiosError(error, 'That code is incorrect. Try again.')
  }
}

export async function regenerateRecoveryCodes(
  request: EnableMfaRequest
): Promise<RecoveryCodesResult> {
  try {
    const { data } = await getApi().post('/auth/mfa/recovery-codes', { code: request.code })
    return { recoveryCodes: data.recovery_codes ?? null }
  } catch (error) {
    throw mapAxiosError(error, 'That code is incorrect. Try again.')
  }
}
