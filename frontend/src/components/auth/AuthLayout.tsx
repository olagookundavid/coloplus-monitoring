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
          <Logo tone="black" className="h-7" />
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

      <div className="relative hidden flex-col items-center justify-center gap-6 overflow-hidden bg-white px-10 py-10 text-center md:flex md:min-h-dvh">
        {/* Decorative only -- soft, blurred brand-teal and near-white bubbles
            behind the lockup, never content. */}
        <div className="pointer-events-none absolute inset-0" aria-hidden="true">
          <div className="absolute -top-24 -left-20 h-80 w-80 rounded-full bg-primary-100 blur-3xl" />
          <div className="absolute top-1/4 -right-24 h-96 w-96 rounded-full bg-primary/25 blur-3xl" />
          <div className="absolute -bottom-28 left-1/4 h-96 w-96 rounded-full bg-primary-50 blur-3xl" />
          <div className="absolute right-8 bottom-8 h-64 w-64 rounded-full bg-primary/15 blur-3xl" />
        </div>

        <div className="relative flex flex-col items-center gap-6">
          <Logo tone="black" variant="full" className="h-48 lg:h-64" />
          <p className="text-sm font-semibold uppercase tracking-[0.25em] text-primary-600">
            Remote monitoring portal
          </p>
        </div>
      </div>
    </div>
  )
}
