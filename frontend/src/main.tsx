import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

import { App } from './App'
import { ToastViewport } from './components/ui'
import { AuthProvider } from './contexts'
import './index.css'

const queryClient = new QueryClient()

// Reload once if a deployed chunk no longer exists (stale client after a
// new deploy) — architecture standard §4 resilience rule.
window.addEventListener('vite:preloadError', () => {
  window.location.reload()
})

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element #root not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <App />
          <ToastViewport />
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>
)
