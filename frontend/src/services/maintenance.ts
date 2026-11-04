import type {
  MaintenanceFilter,
  MaintenanceListResult,
  MaintenanceRules,
  MaintenanceStatus,
  MaintenanceWindow,
} from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface MaintenanceWindowDTO {
  id: string
  site_id: string
  starts_at: string
  ends_at: string
  minutes: number
  reason: string
  status: MaintenanceStatus
  requested_by?: string
  requested_at: string
  decided_by?: string
  decided_at?: string
  decision_note?: string
}

interface MaintenanceRulesDTO {
  notice_hours: number
  max_hours: number
  monthly_cap_hours: number
}

interface MaintenanceListResponseDTO {
  windows: MaintenanceWindowDTO[]
  rules: MaintenanceRulesDTO
  total: number
  limit: number
  offset: number
}

function mapMaintenanceWindow(dto: MaintenanceWindowDTO): MaintenanceWindow {
  return {
    id: dto.id,
    siteId: dto.site_id,
    startsAt: dto.starts_at,
    endsAt: dto.ends_at,
    minutes: dto.minutes,
    reason: dto.reason,
    status: dto.status,
    requestedBy: dto.requested_by,
    requestedAt: dto.requested_at,
    decidedBy: dto.decided_by,
    decidedAt: dto.decided_at,
    decisionNote: dto.decision_note,
  }
}

function mapMaintenanceRules(dto: MaintenanceRulesDTO): MaintenanceRules {
  return {
    noticeHours: dto.notice_hours,
    maxHours: dto.max_hours,
    monthlyCapHours: dto.monthly_cap_hours,
  }
}

export async function fetchMaintenanceWindows(
  filter: MaintenanceFilter
): Promise<MaintenanceListResult> {
  try {
    const { data } = await getApi().get<MaintenanceListResponseDTO>('/maintenance', {
      params: {
        site_id: filter.siteId,
        status: filter.status,
        from: filter.from,
        to: filter.to,
        limit: filter.limit,
        offset: filter.offset,
      },
    })
    return {
      windows: data.windows.map(mapMaintenanceWindow),
      rules: mapMaintenanceRules(data.rules),
      total: data.total,
      limit: data.limit,
      offset: data.offset,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load planned maintenance.')
  }
}

export interface RequestMaintenanceInput {
  startsAt: string
  endsAt: string
  reason: string
}

export async function requestMaintenance(
  siteId: string,
  input: RequestMaintenanceInput
): Promise<MaintenanceWindow> {
  try {
    const { data } = await getApi().post<MaintenanceWindowDTO>(`/sites/${siteId}/maintenance`, {
      starts_at: input.startsAt,
      ends_at: input.endsAt,
      reason: input.reason,
    })
    return mapMaintenanceWindow(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not request this maintenance window.')
  }
}

export async function approveMaintenance(id: string, note?: string): Promise<MaintenanceWindow> {
  try {
    const { data } = await getApi().post<MaintenanceWindowDTO>(`/maintenance/${id}/approve`, {
      note,
    })
    return mapMaintenanceWindow(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not approve this window.')
  }
}

export async function rejectMaintenance(id: string, note: string): Promise<MaintenanceWindow> {
  try {
    const { data } = await getApi().post<MaintenanceWindowDTO>(`/maintenance/${id}/reject`, {
      note,
    })
    return mapMaintenanceWindow(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not reject this window.')
  }
}

export async function cancelMaintenance(id: string, note: string): Promise<MaintenanceWindow> {
  try {
    const { data } = await getApi().post<MaintenanceWindowDTO>(`/maintenance/${id}/cancel`, {
      note,
    })
    return mapMaintenanceWindow(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not cancel this window.')
  }
}
