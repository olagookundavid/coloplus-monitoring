export interface IssuedExport {
  id: string
  kind: string
  filename: string
  params: Record<string, unknown>
  /** Both present when `params` scopes the export to a single site. */
  siteId?: string
  siteName?: string
  rowCount: number
  sizeBytes: number
  sha256: string
  /** False if only the fingerprint was recorded -- the exact bytes weren't
      kept, so the download link can never be reissued for this row. */
  stored: boolean
  issuedBy: string
  issuedByName?: string
  createdAt: string
}

export interface IssuedExportListResult {
  exports: IssuedExport[]
  total: number
  limit: number
  offset: number
}

export interface IssuedExportDownload {
  url: string
  expiresAt: string
  filename: string
}
