import { Logo } from '../components/icons'
import { ButtonLink } from '../components/ui'
import { ROUTES } from '../constants'

export function NotFound() {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-4 bg-surface-subtle px-4 text-center">
      <Logo variant="mark" className="h-8 w-8" />
      <h1 className="text-2xl font-semibold text-text">Page not found</h1>
      <p className="max-w-sm text-sm text-text-muted">
        The page you&rsquo;re looking for doesn&rsquo;t exist or you don&rsquo;t have access to it.
      </p>
      <ButtonLink to={ROUTES.HOME}>Back to home</ButtonLink>
    </div>
  )
}
