import type {
  AcceptInviteRequest,
  AuthApiErrorShape,
  AuthSession,
  ForgotPasswordRequest,
  LoginRequest,
} from '../types'

/**
 * Mock-backed auth service (docs/DECISIONS.md 039, plan "Full UI now, mocked
 * locally"). The backend has no auth feature yet — no User domain type, no
 * migration, no route (see the session's backend inventory). These functions
 * keep the exact call signatures the real axios calls will need, so the
 * eventual swap replaces only the bodies below, never the hooks or pages
 * that call them.
 */

export class AuthApiError extends Error implements AuthApiErrorShape {
  status: number
  fieldErrors?: Record<string, string>

  constructor(shape: AuthApiErrorShape) {
    super(shape.message)
    this.name = 'AuthApiError'
    this.status = shape.status
    this.fieldErrors = shape.fieldErrors
  }
}

interface MockUser {
  id: string
  email: string
  password: string
  name: string
}

const MOCK_USERS: MockUser[] = [
  { id: 'usr_demo', email: 'demo@coloplus.africa', password: 'coloplus123', name: 'Demo User' },
]

function mockDelay(minMs = 500, maxMs = 1000): Promise<void> {
  const duration = minMs + Math.random() * (maxMs - minMs)
  return new Promise((resolve) => setTimeout(resolve, duration))
}

function createMockSession(user: MockUser): AuthSession {
  const expiresAt = new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString()
  return {
    user: { id: user.id, email: user.email, name: user.name },
    accessToken: `mock.${user.id}.${Date.now()}`,
    expiresAt,
  }
}

export async function login(request: LoginRequest): Promise<AuthSession> {
  await mockDelay()
  const match = MOCK_USERS.find(
    (candidate) =>
      candidate.email.toLowerCase() === request.email.trim().toLowerCase() &&
      candidate.password === request.password
  )
  if (!match) {
    throw new AuthApiError({ message: 'Incorrect email or password.', status: 401 })
  }
  return createMockSession(match)
}

export async function requestPasswordReset(request: ForgotPasswordRequest): Promise<void> {
  await mockDelay()
  if (request.email.trim().toLowerCase() === 'error@test.com') {
    throw new AuthApiError({
      message: 'Something went wrong sending the reset link. Try again shortly.',
      status: 500,
    })
  }
  // Intentionally succeeds for any syntactically valid email, matching
  // regardless of account match — never reveal which emails exist
  // (frontend/CLAUDE.md §14).
}

export async function acceptInvite(request: AcceptInviteRequest): Promise<void> {
  await mockDelay()
  if (request.token === 'expired') {
    throw new AuthApiError({
      message: 'This invite link has expired. Ask your administrator to resend it.',
      status: 410,
    })
  }
  // Mock accepts any other non-empty token. Does not return a session:
  // the page never auto-logs the user in on success (deliberate choice).
}
