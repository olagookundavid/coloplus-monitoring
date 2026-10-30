import type { ReactNode } from 'react'

export interface AuthLayoutProps {
  eyebrow?: string
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}

// Each maps to a real, shipped rule, not marketing filler: site scope per
// role, every disputed figure traces back to a report/interval, and the
// append-only audit log.
const BRAND_HIGHLIGHTS = ['Role-based access', 'Evidence-linked records', 'Audit history']

export function AuthLayout({ eyebrow, title, subtitle, children, footer }: AuthLayoutProps) {
  return (
    <div className="min-h-dvh w-full bg-surface md:grid md:grid-cols-2">
      <div className="flex min-h-dvh flex-col px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-2 pt-6">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-sm font-bold text-white">
            C
          </span>
          <span className="text-lg font-bold tracking-tight text-text">Coloplus</span>
        </div>

        <div className="flex flex-1 items-center justify-center py-8 md:max-h-dvh md:overflow-y-auto">
          <div className="w-full max-w-md">
            {eyebrow && <p className="text-sm font-medium text-primary-600">{eyebrow}</p>}
            <h1 className="mt-1 font-heading text-3xl font-extrabold text-text">{title}</h1>
            {subtitle && <p className="mt-2 text-sm text-text-muted">{subtitle}</p>}
            <div className="mt-6">{children}</div>
            {footer && <div className="mt-6 text-sm text-text-muted">{footer}</div>}
          </div>
        </div>

        <p className="pb-6 text-xs text-text-muted">&copy; {new Date().getFullYear()} Coloplus</p>
      </div>

      <div className="auth-brand-panel hidden flex-col justify-center gap-6 px-10 py-10 text-white md:flex md:min-h-dvh">
        <p className="text-xs font-semibold tracking-wide text-white/70 uppercase">
          Remote Monitoring Portal
        </p>

        <div className="flex aspect-[4/3] w-full max-w-md items-center justify-center rounded-lg border border-dashed border-white/30 p-6 text-center text-sm text-white/60">
          Placeholder — site photo or brand imagery goes here
        </div>

        <div>
          <h2 className="font-heading text-3xl font-extrabold text-balance">
            One traceable record of site power service.
          </h2>
          <p className="mt-3 max-w-md text-sm text-white/80">
            Latest site status, fault history and restoration updates, with the evidence behind
            every figure.
          </p>
        </div>

        <div className="flex flex-wrap gap-x-6 gap-y-2 text-xs font-medium text-white/70">
          {BRAND_HIGHLIGHTS.map((label) => (
            <span key={label}>{label}</span>
          ))}
        </div>
      </div>
    </div>
  )
}
