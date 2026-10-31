/* eslint-disable react-refresh/only-export-components -- Provider and its
   useAuth() hook are intentionally colocated in one context file. */
import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { flushSync } from 'react-dom'

import { fetchMe, logout as logoutRequest } from '../services'
import type { MeResult, User } from '../types'

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

interface SessionState {
  user: User | null
  permissions: string[]
  allSites: boolean
  siteIds: string[]
  mfaSetupRequired: boolean
  recoveryCodesLeft: number
}

interface AuthContextValue extends SessionState {
  status: AuthStatus
  isAuthenticated: boolean
  // Returns the freshly-fetched session (or null) so a caller that needs it
  // right away -- e.g. to pick a permission-appropriate landing page --
  // doesn't have to wait for a re-render to see it on context. The state
  // update is flushed synchronously (flushSync) before this resolves, so a
  // navigate() called right after is guaranteed to render against the new
  // session instead of racing a not-yet-committed one -- the bug that sent a
  // freshly-logged-in admin to the customer portal on a cold page load.
  refreshSession: () => Promise<MeResult | null>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

const EMPTY_PERMISSIONS: string[] = []
const EMPTY_SITE_IDS: string[] = []

function emptySession(): SessionState {
  return {
    user: null,
    permissions: EMPTY_PERMISSIONS,
    allSites: false,
    siteIds: EMPTY_SITE_IDS,
    mfaSetupRequired: false,
    recoveryCodesLeft: 0,
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [session, setSession] = useState<SessionState>(() => emptySession())
  const initialized = useRef(false)

  // The session cookie is HttpOnly, so the client has no way to know
  // whether one exists without asking the server — /auth/me is the one
  // source of truth, called once on mount and again after every login.
  async function refreshSession(): Promise<MeResult | null> {
    try {
      const me = await fetchMe()
      flushSync(() => {
        setSession({
          user: me.user,
          permissions: me.permissions,
          allSites: me.allSites,
          siteIds: me.siteIds,
          mfaSetupRequired: me.mfaSetupRequired,
          recoveryCodesLeft: me.recoveryCodesLeft,
        })
        setStatus('authenticated')
      })
      return me
    } catch {
      flushSync(() => {
        setSession(emptySession())
        setStatus('unauthenticated')
      })
      return null
    }
  }

  useEffect(() => {
    if (initialized.current) return
    initialized.current = true
    void refreshSession()
  }, [])

  async function logout() {
    try {
      await logoutRequest()
    } catch {
      // A failed network call shouldn't trap the user signed in locally —
      // clear local state regardless.
    } finally {
      setSession(emptySession())
      setStatus('unauthenticated')
    }
  }

  const value: AuthContextValue = {
    status,
    ...session,
    isAuthenticated: status === 'authenticated',
    refreshSession,
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
