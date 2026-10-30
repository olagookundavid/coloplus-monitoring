/* eslint-disable react-refresh/only-export-components -- Provider and its
   useAuth() hook are intentionally colocated in one context file. */
import { createContext, useContext, useState, type ReactNode } from 'react'

import { AUTH_STORAGE_KEY } from '../constants'
import type { AuthSession, User } from '../types'

interface AuthContextValue {
  user: User | null
  session: AuthSession | null
  isAuthenticated: boolean
  setSession: (session: AuthSession) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

function readStoredSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(AUTH_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as AuthSession
    if (new Date(parsed.expiresAt).getTime() <= Date.now()) {
      localStorage.removeItem(AUTH_STORAGE_KEY)
      return null
    }
    // A session stored before `User.name` existed won't have it — treat it
    // the same as expired rather than handing components a half-shaped
    // user, so a stale browser session bounces back to login instead of
    // crashing the first component that reads `user.name`.
    if (!parsed.user?.name) {
      localStorage.removeItem(AUTH_STORAGE_KEY)
      return null
    }
    return parsed
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSessionState] = useState<AuthSession | null>(() => readStoredSession())

  const setSession = (next: AuthSession) => {
    setSessionState(next)
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(next))
  }

  const logout = () => {
    setSessionState(null)
    localStorage.removeItem(AUTH_STORAGE_KEY)
  }

  const value: AuthContextValue = {
    user: session?.user ?? null,
    session,
    isAuthenticated: session !== null,
    setSession,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}
