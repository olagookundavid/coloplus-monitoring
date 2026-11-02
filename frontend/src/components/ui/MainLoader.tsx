import { Logo } from '../icons'

// App-level loading only: auth bootstrap (RequireAuth, PublicOnlyRoute,
// RequireRole, RequireMfaSetup) and the Suspense fallback around each lazy
// portal layout in App.tsx. Never rendered inside a page -- that's the
// Skeleton system (components/skeletons/).
export function MainLoader() {
  return (
    <div
      role="status"
      aria-label="Loading"
      className="fixed inset-0 z-50 flex items-center justify-center overflow-hidden bg-surface"
    >
      <div className="pointer-events-none absolute inset-0" aria-hidden="true">
        <div className="absolute -top-20 -left-16 h-64 w-64 rounded-full bg-primary-100 blur-3xl" />
        <div className="absolute -right-16 -bottom-20 h-64 w-64 rounded-full bg-primary-50 blur-3xl" />
      </div>
      <Logo tone="black" variant="full" className="relative h-24 animate-pulse-fade sm:h-32" />
    </div>
  )
}
