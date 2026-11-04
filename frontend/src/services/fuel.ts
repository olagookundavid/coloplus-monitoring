import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  FuelEntry,
  FuelEntryKind,
  FuelEstimate,
  FuelStatus,
  FuelVariance,
  RecordFuelInput,
} from '../types'

/**
 * Real backend: GET/POST /sites/:id/fuel, POST /fuel/:id/void
 * (backend/internal/dto/fuel.go). The estimate is always labelled "estimate"
 * -- never a fabricated number when unavailable.
 */

interface FuelVarianceDTO {
  estimated_before_litres: number
  reading_litres: number
  difference_litres: number
  provisional: boolean
  review_warning: boolean
}

interface FuelEntryDTO {
  id: string
  site_id: string
  kind: FuelEntryKind
  litres: number
  occurred_at: string
  note?: string
  recorded_by?: string
  voided?: boolean
  void_reason?: string
  created_at: string
  variance?: FuelVarianceDTO
}

interface FuelEstimateDTO {
  basis: 'estimate'
  available: boolean
  reason?: string
  litres: number | null
  hours_remaining: number | null
  consumption_lph: number
  anchor_kind?: string
  anchor_at?: string
  assumed_empty_start: boolean
  hours_run: number
  consumed_litres: number
  unreported_days: number
  provisional: boolean
  estimated_empty: boolean
  over_capacity: boolean
  tank_capacity_litres: number | null
  as_of: string
  formula: string
  formula_version: string
  explanation: string
  low_fuel_hours: number
  low: boolean
}

interface FuelStatusResponseDTO {
  site_id: string
  estimate: FuelEstimateDTO
  entries: FuelEntryDTO[]
}

function mapVariance(dto?: FuelVarianceDTO): FuelVariance | undefined {
  if (!dto) return undefined
  return {
    estimatedBeforeLitres: dto.estimated_before_litres,
    readingLitres: dto.reading_litres,
    differenceLitres: dto.difference_litres,
    provisional: dto.provisional,
    reviewWarning: dto.review_warning,
  }
}

function mapEntry(dto: FuelEntryDTO): FuelEntry {
  return {
    id: dto.id,
    siteId: dto.site_id,
    kind: dto.kind,
    litres: dto.litres,
    occurredAt: dto.occurred_at,
    note: dto.note,
    recordedBy: dto.recorded_by,
    voided: dto.voided ?? false,
    voidReason: dto.void_reason,
    createdAt: dto.created_at,
    variance: mapVariance(dto.variance),
  }
}

function mapEstimate(dto: FuelEstimateDTO): FuelEstimate {
  return {
    basis: dto.basis,
    available: dto.available,
    reason: dto.reason,
    litres: dto.litres,
    hoursRemaining: dto.hours_remaining,
    consumptionLPH: dto.consumption_lph,
    anchorKind: dto.anchor_kind,
    anchorAt: dto.anchor_at,
    assumedEmptyStart: dto.assumed_empty_start,
    hoursRun: dto.hours_run,
    consumedLitres: dto.consumed_litres,
    unreportedDays: dto.unreported_days,
    provisional: dto.provisional,
    estimatedEmpty: dto.estimated_empty,
    overCapacity: dto.over_capacity,
    tankCapacityLitres: dto.tank_capacity_litres,
    asOf: dto.as_of,
    formula: dto.formula,
    formulaVersion: dto.formula_version,
    explanation: dto.explanation,
    lowFuelHours: dto.low_fuel_hours,
    low: dto.low,
  }
}

export async function fetchFuelStatus(siteId: string): Promise<FuelStatus> {
  try {
    const { data } = await getApi().get<FuelStatusResponseDTO>(`/sites/${siteId}/fuel`)
    return {
      siteId: data.site_id,
      estimate: mapEstimate(data.estimate),
      entries: data.entries.map(mapEntry),
    }
  } catch (error) {
    throw mapAxiosError(error, "Could not load this site's fuel record.")
  }
}

export async function recordFuel(input: RecordFuelInput): Promise<FuelEntry> {
  try {
    const { data } = await getApi().post<FuelEntryDTO>(`/sites/${input.siteId}/fuel`, {
      kind: input.kind,
      litres: input.litres,
      occurred_at: input.occurredAt,
      note: input.note,
      client_key: input.clientKey,
    })
    return mapEntry(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not record this entry.')
  }
}

export async function voidFuelEntry(id: string, reason: string): Promise<FuelEntry> {
  try {
    const { data } = await getApi().post<FuelEntryDTO>(`/fuel/${id}/void`, { reason })
    return mapEntry(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not void this entry.')
  }
}
