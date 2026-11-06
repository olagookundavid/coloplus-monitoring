import type {
  CalculationRun,
  CalculationRunListResult,
  CalculationRunRow,
  CreateCalculationRunInput,
  RunChange,
  RunComparison,
} from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface CalculationRunRowDTO {
  site_id: string
  airtel_code: string
  site_name: string
  eligible_minutes: number
  served_minutes: number
  outage_minutes: number
  unknown_minutes: number
  assumed_minutes: number
  days_in_period: number
  days_reported: number
  reported_availability_percent: number | null
  confirmed_served_share_percent: number | null
  coverage_percent: number | null
  generator_hours: number
  incidents_started: number
  reports_fingerprint: string
}

interface CalculationRunDTO {
  id: string
  kind: string
  period_from: string
  period_to: string
  region: string
  label: string
  formula: string
  formula_version: string
  site_count: number
  created_by: string
  created_by_name?: string
  created_at: string
  rows?: CalculationRunRowDTO[]
}

interface CalculationRunListResponseDTO {
  runs: CalculationRunDTO[]
  total: number
  limit: number
  offset: number
}

interface RunChangeDTO {
  site_id: string
  airtel_code: string
  site_name: string
  then?: CalculationRunRowDTO
  now?: CalculationRunRowDTO
  reports_changed: boolean
}

interface RunComparisonDTO {
  run: CalculationRunDTO
  generated_at: string
  formula_changed: boolean
  current_formula_version: string
  unchanged: number
  changes: RunChangeDTO[]
}

function mapCalculationRunRow(dto: CalculationRunRowDTO): CalculationRunRow {
  return {
    siteId: dto.site_id,
    airtelCode: dto.airtel_code,
    siteName: dto.site_name,
    eligibleMinutes: dto.eligible_minutes,
    servedMinutes: dto.served_minutes,
    outageMinutes: dto.outage_minutes,
    unknownMinutes: dto.unknown_minutes,
    assumedMinutes: dto.assumed_minutes,
    daysInPeriod: dto.days_in_period,
    daysReported: dto.days_reported,
    reportedAvailabilityPercent: dto.reported_availability_percent,
    confirmedServedSharePercent: dto.confirmed_served_share_percent,
    coveragePercent: dto.coverage_percent,
    generatorHours: dto.generator_hours,
    incidentsStarted: dto.incidents_started,
    reportsFingerprint: dto.reports_fingerprint,
  }
}

function mapCalculationRun(dto: CalculationRunDTO): CalculationRun {
  return {
    id: dto.id,
    kind: dto.kind,
    periodFrom: dto.period_from,
    periodTo: dto.period_to,
    region: dto.region,
    label: dto.label,
    formula: dto.formula,
    formulaVersion: dto.formula_version,
    siteCount: dto.site_count,
    createdBy: dto.created_by,
    createdByName: dto.created_by_name,
    createdAt: dto.created_at,
    rows: dto.rows?.map(mapCalculationRunRow),
  }
}

function mapRunChange(dto: RunChangeDTO): RunChange {
  return {
    siteId: dto.site_id,
    airtelCode: dto.airtel_code,
    siteName: dto.site_name,
    then: dto.then ? mapCalculationRunRow(dto.then) : undefined,
    now: dto.now ? mapCalculationRunRow(dto.now) : undefined,
    reportsChanged: dto.reports_changed,
  }
}

export async function createCalculationRun(
  input: CreateCalculationRunInput
): Promise<CalculationRun> {
  try {
    const { data } = await getApi().post<CalculationRunDTO>('/calculation-runs', {
      month: input.month,
      region: input.region,
      label: input.label,
    })
    return mapCalculationRun(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not store this run.')
  }
}

export async function fetchCalculationRuns(params: {
  limit?: number
  offset?: number
}): Promise<CalculationRunListResult> {
  try {
    const { data } = await getApi().get<CalculationRunListResponseDTO>('/calculation-runs', {
      params,
    })
    return {
      runs: data.runs.map(mapCalculationRun),
      total: data.total,
      limit: data.limit,
      offset: data.offset,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load calculation runs.')
  }
}

export async function fetchCalculationRun(id: string): Promise<CalculationRun> {
  try {
    const { data } = await getApi().get<CalculationRunDTO>(`/calculation-runs/${id}`)
    return mapCalculationRun(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this run.')
  }
}

export async function fetchCalculationRunComparison(id: string): Promise<RunComparison> {
  try {
    const { data } = await getApi().get<RunComparisonDTO>(`/calculation-runs/${id}/compare`)
    return {
      run: mapCalculationRun(data.run),
      generatedAt: data.generated_at,
      formulaChanged: data.formula_changed,
      currentFormulaVersion: data.current_formula_version,
      unchanged: data.unchanged,
      changes: data.changes.map(mapRunChange),
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not compare this run to today.')
  }
}
