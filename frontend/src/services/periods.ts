import type { PeriodClosure } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface PeriodClosureDTO {
  id: string
  month: string
  closed_by: string
  closed_by_name?: string
  closed_at: string
  run_id: string
  locked: boolean
  reopened_by?: string
  reopened_by_name?: string
  reopened_at?: string
  reopen_reason?: string
}

function mapPeriodClosure(dto: PeriodClosureDTO): PeriodClosure {
  return {
    id: dto.id,
    month: dto.month,
    closedBy: dto.closed_by,
    closedByName: dto.closed_by_name,
    closedAt: dto.closed_at,
    runId: dto.run_id,
    locked: dto.locked,
    reopenedBy: dto.reopened_by,
    reopenedByName: dto.reopened_by_name,
    reopenedAt: dto.reopened_at,
    reopenReason: dto.reopen_reason,
  }
}

export async function fetchPeriods(): Promise<PeriodClosure[]> {
  try {
    const { data } = await getApi().get<{ periods: PeriodClosureDTO[] }>('/periods')
    return data.periods.map(mapPeriodClosure)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load closed periods.')
  }
}

// No body -- the grace period and "already closed" rules are enforced
// server-side and surface as the request's own error message.
export async function closePeriod(month: string): Promise<PeriodClosure> {
  try {
    const { data } = await getApi().post<PeriodClosureDTO>(`/periods/${month}/close`, {})
    return mapPeriodClosure(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not close this month.')
  }
}

export async function reopenPeriod(month: string, reason: string): Promise<PeriodClosure> {
  try {
    const { data } = await getApi().post<PeriodClosureDTO>(`/periods/${month}/reopen`, { reason })
    return mapPeriodClosure(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not reopen this month.')
  }
}
