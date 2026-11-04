import { getApi } from './api'
import { mapAxiosError } from './auth'
import { downloadCsv } from '../utils/csv'

export interface ExportDateRange {
  from: string
  to: string
  region?: string
  siteId?: string
}

async function downloadExport(
  path: string,
  filename: string,
  params: Record<string, string | undefined>
) {
  try {
    const { data } = await getApi().get<string>(path, { params, responseType: 'text' })
    downloadCsv(filename, data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not generate this export.')
  }
}

export function downloadDailySummariesCsv(range: ExportDateRange): Promise<void> {
  return downloadExport('/exports/daily-summaries.csv', 'coloplus-daily-summaries.csv', {
    from: range.from,
    to: range.to,
    region: range.region,
    site_id: range.siteId,
  })
}

export function downloadIncidentRegisterCsv(range: ExportDateRange): Promise<void> {
  return downloadExport('/exports/incidents.csv', 'coloplus-incident-register.csv', {
    from: range.from,
    to: range.to,
    region: range.region,
    site_id: range.siteId,
  })
}

export interface MonthlyPerformanceParams {
  /** "YYYY-MM" */
  month: string
  region?: string
}

export function downloadMonthlyPerformanceCsv(params: MonthlyPerformanceParams): Promise<void> {
  return downloadExport(
    '/exports/monthly-performance.csv',
    `coloplus-monthly-performance_${params.month}.csv`,
    { month: params.month, region: params.region }
  )
}

export interface WeeklyExceptionsParams {
  /** "YYYY-MM-DD", must be a Monday -- callers snap to one with mondayOf(). */
  weekStart: string
  region?: string
}

export function downloadWeeklyExceptionsCsv(params: WeeklyExceptionsParams): Promise<void> {
  return downloadExport(
    '/exports/weekly-exceptions.csv',
    `coloplus-weekly-exceptions_${params.weekStart}.csv`,
    { week_start: params.weekStart, region: params.region }
  )
}

export interface CommissioningBacklogParams {
  region?: string
}

export function downloadCommissioningBacklogCsv(
  params: CommissioningBacklogParams = {}
): Promise<void> {
  return downloadExport(
    '/exports/commissioning-backlog.csv',
    'coloplus-commissioning-backlog.csv',
    { region: params.region }
  )
}
