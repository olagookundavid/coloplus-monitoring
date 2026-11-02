import { getApi } from './api'
import { mapAxiosError } from './auth'
import type { IncidentCorrection, IncidentCorrectionField } from '../types'

interface IncidentCorrectionDTO {
  id: string
  ref: string
  incident_id: string
  site_id: string
  field: IncidentCorrectionField
  original_value: string
  corrected_value: string
  reason: string
  status: IncidentCorrection['status']
  requested_by: string
  reviewed_by?: string
  reviewed_at?: string
  review_note?: string
  created_at: string
  updated_at: string
}

interface IncidentCorrectionListResponseDTO {
  corrections?: IncidentCorrectionDTO[]
  total: number
}

function mapCorrection(dto: IncidentCorrectionDTO): IncidentCorrection {
  return {
    id: dto.id,
    ref: dto.ref,
    incidentId: dto.incident_id,
    siteId: dto.site_id,
    field: dto.field,
    originalValue: dto.original_value,
    correctedValue: dto.corrected_value,
    reason: dto.reason,
    status: dto.status,
    requestedBy: dto.requested_by,
    reviewedBy: dto.reviewed_by,
    reviewedAt: dto.reviewed_at,
    reviewNote: dto.review_note,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

// One call, no status filter -- same "bucket client-side" precedent as
// fetchReporterIncidents, a reporter's own correction-request volume is small.
export async function fetchIncidentCorrections(): Promise<IncidentCorrection[]> {
  try {
    const { data } = await getApi().get<IncidentCorrectionListResponseDTO>('/incident-corrections', {
      params: { limit: 200 },
    })
    return (data.corrections ?? []).map(mapCorrection)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load corrections.')
  }
}

export async function requestIncidentCorrection(
  incidentId: string,
  field: IncidentCorrectionField,
  correctedValue: string,
  reason: string
): Promise<IncidentCorrection> {
  try {
    const { data } = await getApi().post<IncidentCorrectionDTO>(`/incidents/${incidentId}/corrections`, {
      field,
      corrected_value: correctedValue,
      reason,
    })
    return mapCorrection(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not send this correction request.')
  }
}
