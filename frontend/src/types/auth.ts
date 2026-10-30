export interface User {
  id: string
  email: string
  name: string
}

export interface AuthSession {
  user: User
  accessToken: string
  expiresAt: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface ForgotPasswordRequest {
  email: string
}

export interface AcceptInviteRequest {
  token: string
  password: string
}

export interface AuthApiErrorShape {
  message: string
  status: number
  fieldErrors?: Record<string, string>
}
