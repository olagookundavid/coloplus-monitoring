import type { StatusTone } from '../components/ui'
import type { SiteImportRowAction } from '../types'

// Mirrors backend/internal/services/site_import.go's own cap -- checked
// client-side so an oversized file never reaches the network.
export const MAX_SITE_IMPORT_BYTES = 800 * 1024

/** A client-side pre-check mirroring the backend's own validation, so the
    error shows up immediately rather than after a network round trip. */
export function validateSiteImportFile(file: File): string | undefined {
  if (file.size <= 0) return 'This file is empty.'
  if (file.size > MAX_SITE_IMPORT_BYTES) {
    return 'This file is larger than 800 KB.'
  }
  if (!file.name.toLowerCase().endsWith('.csv')) {
    return 'Choose a CSV file.'
  }
  return undefined
}

export function siteImportRowActionLabel(action: SiteImportRowAction): {
  label: string
  tone: StatusTone
} {
  switch (action) {
    case 'create':
      return { label: 'Create', tone: 'success' }
    case 'update':
      return { label: 'Update', tone: 'info' }
    case 'unchanged':
      return { label: 'Unchanged', tone: 'neutral' }
    case 'error':
      return { label: 'Error', tone: 'danger' }
  }
}
