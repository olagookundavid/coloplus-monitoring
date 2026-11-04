export type SiteImportRowAction = 'create' | 'update' | 'unchanged' | 'error'

export interface SiteImportRow {
  line: number
  airtelCode: string
  action: SiteImportRowAction
  problems?: string[]
}

export interface SiteImportResult {
  dryRun: boolean
  applied: boolean
  created: number
  updated: number
  unchanged: number
  rejected: number
  rows: SiteImportRow[]
}
