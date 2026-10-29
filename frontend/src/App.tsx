import { Suspense } from 'react'
import { Route, Routes } from 'react-router-dom'

// Direct import, not the top-level `./components` barrel: that barrel
// re-exports components/dashboard (Recharts, TanStack Table), and App.tsx
// is eager — importing anything through it would pull those into the main
// bundle instead of the lazy customer-portal chunk.
import { ErrorBoundary } from './components/ErrorBoundary'
import { PublicOnlyRoute, RequireAuth } from './components/auth'
import { Spinner } from './components/ui'
import { ROUTES } from './constants'
import { NotFound } from './pages'
import { AcceptInvite, ForgotPassword, Login } from './pages/auth'
import type { ComingSoonProps } from './pages/customer/ComingSoon'
import { lazyNamed } from './utils'

// This codebase's pages use named exports; pages/customer/ is the lazy
// route group (frontend/CLAUDE.md §9 — pages/auth/ is the eager "landing"
// group, customer/ops are lazy-loaded).
const CustomerLayout = lazyNamed(() => import('./pages/customer/CustomerLayout'), 'CustomerLayout')
const Overview = lazyNamed(() => import('./pages/customer/Overview'), 'Overview')
const ComingSoon = lazyNamed<ComingSoonProps>(
  () => import('./pages/customer/ComingSoon'),
  'ComingSoon'
)

export function App() {
  return (
    <ErrorBoundary>
      <Routes>
        <Route
          path={ROUTES.LOGIN}
          element={
            <PublicOnlyRoute>
              <Login />
            </PublicOnlyRoute>
          }
        />
        <Route
          path={ROUTES.FORGOT_PASSWORD}
          element={
            <PublicOnlyRoute>
              <ForgotPassword />
            </PublicOnlyRoute>
          }
        />
        <Route path={ROUTES.ACCEPT_INVITE} element={<AcceptInvite />} />

        <Route
          element={
            <RequireAuth>
              <Suspense fallback={<Spinner className="mx-auto mt-20 h-6 w-6" />}>
                <CustomerLayout />
              </Suspense>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.HOME} element={<Overview />} />
          <Route path={ROUTES.CUSTOMER_SITES} element={<ComingSoon title="Sites" />} />
          <Route path={ROUTES.SITE_DETAIL_PATTERN} element={<ComingSoon title="Site detail" />} />
          <Route path={ROUTES.CUSTOMER_TICKETS} element={<ComingSoon title="Support tickets" />} />
          <Route path={ROUTES.CUSTOMER_REPORTS} element={<ComingSoon title="Reports" />} />
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
    </ErrorBoundary>
  )
}
