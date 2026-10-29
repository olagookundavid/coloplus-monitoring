import { Component, type ErrorInfo, type ReactNode } from 'react'

import { Button } from './ui'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled error in route tree', error, info)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-dvh flex-col items-center justify-center gap-4 bg-surface-subtle px-4 text-center">
          <p className="text-sm font-medium text-primary-600">Coloplus Monitoring</p>
          <h1 className="text-2xl font-semibold text-text">Something went wrong</h1>
          <p className="max-w-sm text-sm text-text-muted">
            Reload the page. If the problem continues, contact support.
          </p>
          <Button onClick={() => window.location.assign('/')}>Reload</Button>
        </div>
      )
    }
    return this.props.children
  }
}
