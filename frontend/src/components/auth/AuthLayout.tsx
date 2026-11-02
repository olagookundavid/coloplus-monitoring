import type { ReactNode } from 'react'

import { Logo } from '../icons'

export interface AuthLayoutProps {
  eyebrow?: string
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}

export function AuthLayout({ eyebrow, title, subtitle, children, footer }: AuthLayoutProps) {
  return (
    <div className="min-h-dvh w-full bg-surface md:grid md:grid-cols-2">
      <div className="flex min-h-dvh flex-col px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-2 pt-6">
          <Logo className="h-7" />
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

      <div className="auth-brand-panel hidden flex-col items-center justify-center gap-8 px-10 py-10 text-center text-white md:flex md:min-h-dvh">
        <Logo className="h-40 lg:h-56" />
        <p className="max-w-sm text-base text-white/70">
          One traceable record of site power service — latest status, fault history and the evidence
          behind every figure.
        </p>
      </div>
    </div>
  )
}
