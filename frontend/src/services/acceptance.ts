import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  Acceptance,
  AcceptanceEvent,
  AcceptanceParty,
  AcceptanceStage,
  TransitionAcceptanceInput,
} from '../types'

interface AcceptanceDTO {
  id: string
  site_id: string
  stage: AcceptanceStage
  held_from?: AcceptanceStage
  waiting_on: AcceptanceParty
  next_action?: string
  due_at?: string
  overdue: boolean
  dossier?: string
  reason?: string
  hold_reason?: string
  ready_age_hours: number
  requested_at: string
  built_at?: string
  ready_at?: string
  inspected_at?: string
  decided_at?: string
  updated_at: string
}

interface AcceptanceEventDTO {
  id: string
  party: AcceptanceParty
  from_stage?: AcceptanceStage
  to_stage: AcceptanceStage
  body?: string
  created_at: string
}

interface AcceptanceListResponseDTO {
  acceptances?: AcceptanceDTO[]
  total: number
}

export function mapAcceptance(dto: AcceptanceDTO): Acceptance {
  return {
    id: dto.id,
    siteId: dto.site_id,
    stage: dto.stage,
    heldFrom: dto.held_from,
    waitingOn: dto.waiting_on,
    nextAction: dto.next_action,
    dueAt: dto.due_at,
    overdue: dto.overdue,
    dossier: dto.dossier,
    reason: dto.reason,
    holdReason: dto.hold_reason,
    readyAgeHours: dto.ready_age_hours,
    requestedAt: dto.requested_at,
    builtAt: dto.built_at,
    readyAt: dto.ready_at,
    inspectedAt: dto.inspected_at,
    decidedAt: dto.decided_at,
    updatedAt: dto.updated_at,
  }
}

function mapEvent(dto: AcceptanceEventDTO): AcceptanceEvent {
  return {
    id: dto.id,
    party: dto.party,
    fromStage: dto.from_stage,
    toStage: dto.to_stage,
    body: dto.body ?? '',
    createdAt: dto.created_at,
  }
}

export interface AcceptanceFilter {
  stage?: AcceptanceStage
  openOnly?: boolean
  overdue?: boolean
}

export async function fetchAcceptances(filter: AcceptanceFilter = {}): Promise<Acceptance[]> {
  try {
    const { data } = await getApi().get<AcceptanceListResponseDTO>('/acceptances', {
      params: {
        stage: filter.stage,
        open: filter.openOnly ? 'true' : undefined,
        overdue: filter.overdue ? 'true' : undefined,
        limit: 200,
      },
    })
    return (data.acceptances ?? []).map(mapAcceptance)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the acceptance backlog.')
  }
}

export interface RequestAcceptanceInput {
  nextAction?: string
  dueAt?: string
}

export async function requestAcceptance(
  siteId: string,
  input: RequestAcceptanceInput = {}
): Promise<Acceptance> {
  try {
    const { data } = await getApi().post<AcceptanceDTO>(`/sites/${siteId}/acceptance`, {
      next_action: input.nextAction,
      due_at: input.dueAt,
    })
    return mapAcceptance(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not start acceptance for this site.')
  }
}

export async function fetchSiteAcceptance(
  siteId: string
): Promise<{ acceptance: Acceptance; events: AcceptanceEvent[] }> {
  try {
    const { data } = await getApi().get<{
      acceptance: AcceptanceDTO
      events?: AcceptanceEventDTO[]
    }>(`/sites/${siteId}/acceptance`)
    return { acceptance: mapAcceptance(data.acceptance), events: (data.events ?? []).map(mapEvent) }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this site’s acceptance record.')
  }
}

export async function transitionAcceptance(
  siteId: string,
  input: TransitionAcceptanceInput
): Promise<Acceptance> {
  try {
    const { data } = await getApi().post<AcceptanceDTO>(
      `/sites/${siteId}/acceptance/transition`,
      { stage: input.stage, note: input.note }
    )
    return mapAcceptance(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update this acceptance record.')
  }
}
