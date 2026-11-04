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
const CustomerIncidents = lazyNamed(() => import('./pages/customer/Incidents'), 'Incidents')
const Disputes = lazyNamed(() => import('./pages/customer/Disputes'), 'Disputes')
const Acceptance = lazyNamed(() => import('./pages/customer/Acceptance'), 'Acceptance')
const SupportTickets = lazyNamed(() => import('./pages/customer/SupportTickets'), 'SupportTickets')
const Reports = lazyNamed(() => import('./pages/customer/Reports'), 'Reports')

const OpsLayout = lazyNamed(() => import('./pages/ops/OpsLayout'), 'OpsLayout')
const OpsOverview = lazyNamed(() => import('./pages/ops/Overview'), 'Overview')
const OpsUsers = lazyNamed(() => import('./pages/ops/Users'), 'Users')
const OpsSites = lazyNamed(() => import('./pages/ops/Sites'), 'Sites')
const OpsSiteAssignments = lazyNamed(() => import('./pages/ops/SiteAssignments'), 'SiteAssignments')
const OpsApprovals = lazyNamed(() => import('./pages/ops/Approvals'), 'Approvals')
const OpsAuditLog = lazyNamed(() => import('./pages/ops/AuditLog'), 'AuditLog')
const OpsEmailLog = lazyNamed(() => import('./pages/ops/EmailLog'), 'EmailLog')
const OpsSettings = lazyNamed(() => import('./pages/ops/Settings'), 'Settings')
const Periods = lazyNamed(() => import('./pages/ops/Periods'), 'Periods')
const Maintenance = lazyNamed(() => import('./pages/ops/Maintenance'), 'Maintenance')
const IssuedExports = lazyNamed(() => import('./pages/ops/IssuedExports'), 'IssuedExports')
const CalculationRuns = lazyNamed(() => import('./pages/ops/CalculationRuns'), 'CalculationRuns')
const AuditorOverview = lazyNamed(() => import('./pages/ops/AuditorOverview'), 'AuditorOverview')
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

        {/* published.read is the door key both airtel_viewer and
            airtel_reviewer hold -- the one permission unique to the two
            Airtel roles among everyone not already routed to /ops/* by
            landingRouteFor. Reviewer-only actions (raise/comment/withdraw a
            dispute, inspect/accept/reject acceptance) are gated per-button
            inside Disputes/Acceptance, same pattern as Approvals.tsx. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="published.read">
                <Suspense fallback={<MainLoader />}>
                  <CustomerLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.HOME} element={<Overview />} />
          <Route path={ROUTES.CUSTOMER_SITES} element={<Sites />} />
          <Route path={ROUTES.SITE_DETAIL_PATTERN} element={<Sites />} />
          <Route path={ROUTES.CUSTOMER_INCIDENTS} element={<CustomerIncidents />} />
          <Route path={ROUTES.INCIDENT_DETAIL_PATTERN} element={<CustomerIncidents />} />
          <Route path={ROUTES.CUSTOMER_DISPUTES} element={<Disputes />} />
          <Route path={ROUTES.CUSTOMER_ACCEPTANCE} element={<Acceptance />} />
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
          {/* Gated here on users.manage like the rest of this block, not a
              new exports.create block: the backend's real check for
              /calculation-runs is exports.create AND Actor.AllSites AND NOT
              published.read, and only admin currently holds AllSites, so
              users.manage is the permission that actually matches who can
              use this page today (see opsNav.ts for the same note). */}
          <Route path={ROUTES.OPS_CALCULATION_RUNS} element={<CalculationRuns />} />
          <Route path={ROUTES.CALCULATION_RUN_DETAIL_PATTERN} element={<CalculationRuns />} />
        </Route>

        {/* A separate block, not folded into the one above: auditor (sites.read,
            no users.manage) reads the site directory too, read-only (Sites.tsx
            guards every write control on sites.manage internally). */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission={['users.manage', 'sites.read']}>
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_SITES} element={<OpsSites />} />
        </Route>

        {/* A separate block, not folded into the one above: Approvals is the
            one ops page a supervisor (reports.review, no users.manage) and an
            auditor (reports.read, neither) must reach too, alongside an admin
            viewing read-only. Users and site assignments stay admin-only,
            unchanged. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission={['users.manage', 'reports.review', 'reports.read']}>
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_APPROVALS} element={<OpsApprovals />} />
        </Route>

        {/* periods.manage is admin-only today, like users.manage, but kept
            as its own block for the same reason as audit.read/
            notifications.read below: it's the permission that actually
            controls this data, not guaranteed to stay bundled with
            users.manage forever. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="periods.manage">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_PERIODS} element={<Periods />} />
        </Route>

        {/* Another separate block: audit.read is the permission that
            actually controls this data, held by admin today but not
            necessarily bundled with users.manage forever -- the PRD's
            read-only "management/auditor" role now holds audit.read alone,
            and gets its own Overview page bundled into this same block
            (same one-permission-one-block pattern as the reporter/
            supervisor blocks below carrying several routes each). */}
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
          <Route path={ROUTES.OPS_AUDITOR_OVERVIEW} element={<AuditorOverview />} />
        </Route>

        {/* notifications.read is its own permission too, admin-only today
            via the same migration that grants audit.read -- own block
            rather than widening the audit.read list, same reasoning as
            above: the two aren't guaranteed to stay bundled. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="notifications.read">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_EMAIL_LOG} element={<OpsEmailLog />} />
        </Route>

        {/* A fifth, separate block: reports.create is the field reporter's
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

        {/* A sixth, separate block: the operations supervisor portal.
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

        {/* A seventh, separate block: maintenance.request is held by all
            three ops roles (admin, supervisor, reporter) -- the one
            permission that exactly matches "everyone who can at least
            request a maintenance window," distinct from every existing
            list above. maintenance.read (auditor) is widened in here too,
            read-only: Maintenance.tsx guards the "Request maintenance"
            button on maintenance.request internally, and approve/reject/
            cancel are already gated per-row on maintenance.approve or being
            the requester. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission={['maintenance.request', 'maintenance.read']}>
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_MAINTENANCE} element={<Maintenance />} />
        </Route>

        {/* An eighth, separate block: exports.create is held by admin and
            supervisor alike (and by both Airtel roles, who never reach the
            ops portal at all) -- the real permission the backend checks for
            /exports/issued. Who actually sees in it (own exports vs
            everyone's) is decided server-side by audit.read, not by
            anything this route checks. */}
        <Route
          element={
            <RequireAuth>
              <RequireRole permission="exports.create">
                <Suspense fallback={<MainLoader />}>
                  <OpsLayout />
                </Suspense>
              </RequireRole>
            </RequireAuth>
          }
        >
          <Route path={ROUTES.OPS_ISSUED_EXPORTS} element={<IssuedExports />} />
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
    </ErrorBoundary>
  )
}
