import type { ReactNode } from 'react'

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
      <div className="flex items-center justify-center gap-3 bg-gradient-to-br from-primary-700 to-primary-600 px-4 py-6 text-white md:min-h-dvh md:flex-col md:gap-4 md:px-10 md:py-0">
        <span className="text-2xl font-bold tracking-tight md:text-3xl">Coloplus</span>
        <p className="hidden text-sm text-white/80 md:block">
          Remote power monitoring for Airtel sites
        </p>
      </div>

      <div className="flex flex-1 items-center justify-center px-4 py-8 sm:px-6 lg:px-8 md:max-h-dvh md:overflow-y-auto">
        <div className="w-full max-w-md">
          {eyebrow && <p className="text-sm font-medium text-primary-600">{eyebrow}</p>}
          <h1 className="mt-1 text-2xl font-semibold text-text">{title}</h1>
          {subtitle && <p className="mt-2 text-sm text-text-muted">{subtitle}</p>}
          <div className="mt-6">{children}</div>
          {footer && <div className="mt-6 text-sm text-text-muted">{footer}</div>}
        </div>
      </div>
    </div>
  )
}
