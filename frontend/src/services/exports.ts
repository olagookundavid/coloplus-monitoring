import { getApi } from './api'
import { mapAxiosError } from './auth'
import { downloadCsv } from '../utils/csv'

export interface ExportDateRange {
  from: string
  to: string
  region?: string
  siteId?: string
}

async function downloadExport(path: string, filename: string, range: ExportDateRange) {
  try {
    const { data } = await getApi().get<string>(path, {
      params: { from: range.from, to: range.to, region: range.region, site_id: range.siteId },
      responseType: 'text',
    })
    downloadCsv(filename, data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not generate this export.')
  }
}

export function downloadDailySummariesCsv(range: ExportDateRange): Promise<void> {
  return downloadExport('/exports/daily-summaries.csv', 'coloplus-daily-summaries.csv', range)
}

export function downloadIncidentRegisterCsv(range: ExportDateRange): Promise<void> {
  return downloadExport('/exports/incidents.csv', 'coloplus-incident-register.csv', range)
}
