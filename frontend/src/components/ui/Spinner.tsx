import { ArrowPathIcon } from '@heroicons/react/24/outline'

export interface SpinnerProps {
  className?: string
}

export function Spinner({ className = 'h-4 w-4' }: SpinnerProps) {
  return <ArrowPathIcon className={`animate-spin ${className}`} aria-hidden="true" />
}
