import type { SiteImportResult, SiteImportRow, SiteImportRowAction } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface SiteImportRowDTO {
  line: number
  customer_site_code: string
  action: SiteImportRowAction
  problems?: string[]
}

interface SiteImportResultDTO {
  dry_run: boolean
  applied: boolean
  created: number
  updated: number
  unchanged: number
  rejected: number
  rows: SiteImportRowDTO[]
}

function mapSiteImportRow(dto: SiteImportRowDTO): SiteImportRow {
  return {
    line: dto.line,
    customerSiteCode: dto.customer_site_code,
    action: dto.action,
    problems: dto.problems,
  }
}

function mapSiteImportResult(dto: SiteImportResultDTO): SiteImportResult {
  return {
    dryRun: dto.dry_run,
    applied: dto.applied,
    created: dto.created,
    updated: dto.updated,
    unchanged: dto.unchanged,
    rejected: dto.rejected,
    rows: dto.rows.map(mapSiteImportRow),
  }
}

export interface ImportSitesInput {
  csv: string
  dryRun: boolean
}

// The API takes the file's raw text as a JSON string field, not a
// multipart upload -- the frontend reads the picked File with file.text()
// before calling this.
export async function importSitesCsv(input: ImportSitesInput): Promise<SiteImportResult> {
  try {
    const { data } = await getApi().post<SiteImportResultDTO>('/sites/import', {
      csv: input.csv,
      dry_run: input.dryRun,
    })
    return mapSiteImportResult(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not import this file.')
  }
}
