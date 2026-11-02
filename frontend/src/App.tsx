import { Suspense } from 'react'
import { Route, Routes } from 'react-router-dom'

// Direct import, not the top-level `./components` barrel: that barrel
// re-exports components/dashboard (Recharts, TanStack Table), and App.tsx
// is eager — importing anything through it would pull those into the main
// bundle instead of the lazy customer-portal chunk.
import { ErrorBoundary } from './components/ErrorBoundary'
import { PublicOnlyRoute, RequireAuth, RequireMfaSetup, RequireRole } from './components/auth'
import { MainLoader } from './components/ui'
import { ROUTES } from './constants'
import { NotFound } from './pages'
import { AcceptInvite, ForgotPassword, Login, MfaSetup, ResetPassword } from './pages/auth'
import { lazyNamed } from './utils'

// This codebase's pages use named exports; pages/customer/ and pages/ops/
// are the lazy route groups (frontend/CLAUDE.md §9 — pages/auth/ is the
// eager "landing" group).
const CustomerLayout = lazyNamed(() => import('./pages/customer/CustomerLayout'), 'CustomerLayout')
const Overview = lazyNamed(() => import('./pages/customer/Overview'), 'Overview')
const Sites = lazyNamed(() => import('./pages/customer/Sites'), 'Sites')
const SupportTickets = lazyNamed(() => import('./pages/customer/SupportTickets'), 'SupportTickets')
const Reports = lazyNamed(() => import('./pages/customer/Reports'), 'Reports')

const OpsLayout = lazyNamed(() => import('./pages/ops/OpsLayout'), 'OpsLayout')
const OpsOverview = lazyNamed(() => import('./pages/ops/Overview'), 'Overview')
const OpsUsers = lazyNamed(() => import('./pages/ops/Users'), 'Users')
const OpsSiteAssignments = lazyNamed(() => import('./pages/ops/SiteAssignments'), 'SiteAssignments')
const OpsApprovals = lazyNamed(() => import('./pages/ops/Approvals'), 'Approvals')
const OpsAuditLog = lazyNamed(() => import('./pages/ops/AuditLog'), 'AuditLog')
const OpsSettings = lazyNamed(() => import('./pages/ops/Settings'), 'Settings')
const ReporterOverview = lazyNamed(() => import('./pages/ops/ReporterOverview'), 'ReporterOverview')
const MySites = lazyNamed(() => import('./pages/ops/MySites'), 'MySites')
const DailyReports = lazyNamed(() => import('./pages/ops/DailyReports'), 'DailyReports')
const ReporterIncidents = lazyNamed(
  () => import('./pages/ops/ReporterIncidents'),
  'ReporterIncidents'
)
const Corrections = lazyNamed(() => import('./pages/ops/Corrections'), 'Corrections')
const SupervisorOverview = lazyNamed(
  () => import('./pages/ops/SupervisorOverview'),
  'SupervisorOverview'
)
const SupervisorSites = lazyNamed(() => import('./pages/ops/SupervisorSites'), 'SupervisorSites')
const SupervisorIncidents = lazyNamed(
  () => import('./pages/ops/SupervisorIncidents'),
  'SupervisorIncidents'
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
        <Route path={ROUTES.RESET_PASSWORD} element={<ResetPassword />} />
        <Route
          path={ROUTES.MFA_SETUP}
          element={
            <RequireMfaSetup>
              <MfaSetup />
            </RequireMfaSetup>
          }
        />

        <Route
          element={
            <RequireAuth>
              <Suspense fallback={<MainLoader />}>
                <CustomerLayout />
              </Suspense>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.HOME} element={<Overview />} />
          <Route path={ROUTES.CUSTOMER_SITES} element={<Sites />} />
          <Route path={ROUTES.SITE_DETAIL_PATTERN} element={<Sites />} />
          <Route path={ROUTES.CUSTOMER_TICKETS} element={<SupportTickets />} />
          <Route path={ROUTES.TICKET_DETAIL_PATTERN} element={<SupportTickets />} />
          <Route path={ROUTES.CUSTOMER_REPORTS} element={<Reports />} />
        </Route>

        <Route
          element={
            <RequireAuth>
              <RequireRole permission="users.manage">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_OVERVIEW} element={<OpsOverview />} />
          <Route path={ROUTES.OPS_USERS} element={<OpsUsers />} />
          <Route path={ROUTES.OPS_SITE_ASSIGNMENTS} element={<OpsSiteAssignments />} />
          <Route path={ROUTES.OPS_SETTINGS} element={<OpsSettings />} />
        </Route>

        {/* A separate block, not folded into the one above: Approvals is the
            one ops page a supervisor (reports.review, no users.manage) must
            reach too, alongside an admin viewing read-only. Users and site
            assignments stay admin-only, unchanged. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission={['users.manage', 'reports.review']}>
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_APPROVALS} element={<OpsApprovals />} />
        </Route>

        {/* Another separate block: audit.read is the permission that
            actually controls this data, held by admin today but not
            necessarily bundled with users.manage forever (the PRD's
            read-only "management/auditor" role would hold audit.read alone). */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="audit.read">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_AUDIT_LOG} element={<OpsAuditLog />} />
        </Route>

        {/* A fourth, separate block: reports.create is the field reporter's
            own permission, held by no other ops role (App.tsx's existing
            pattern -- Approvals and Audit log each get their own block
            rather than widening an existing permission list). */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="reports.create">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_REPORTER_OVERVIEW} element={<ReporterOverview />} />
          <Route path={ROUTES.OPS_MY_SITES} element={<MySites />} />
          <Route path={ROUTES.OPS_DAILY_REPORTS} element={<DailyReports />} />
          <Route path={ROUTES.OPS_INCIDENTS} element={<ReporterIncidents />} />
          <Route path={ROUTES.OPS_CORRECTIONS} element={<Corrections />} />
        </Route>

        {/* A fifth, separate block: the operations supervisor portal.
            reports.review is the door key (same reasoning as the Approvals
            block above) -- it's the one permission this codebase already
            uses to mean "supervisor-flavored actor," and admin explicitly
            does not hold it (Decision 025), so this cleanly excludes admin
            from a near-duplicate nav group. Finer actions inside Incidents
            (assign/transition/publish) still check incidents.manage. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="reports.review">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_SUPERVISOR_OVERVIEW} element={<SupervisorOverview />} />
          <Route path={ROUTES.OPS_SUPERVISOR_SITES} element={<SupervisorSites />} />
          <Route path={ROUTES.OPS_SUPERVISOR_INCIDENTS} element={<SupervisorIncidents />} />
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
    </ErrorBoundary>
  )
}
