import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  Dispute,
  DisputeCategory,
  DisputeDetail,
  DisputeEvent,
  DisputeEventKind,
  DisputeMetric,
  DisputeOutcome,
  DisputeParty,
  DisputeStatus,
  DisputeSummary,
  RaiseDisputeInput,
} from '../types'

interface DisputeDTO {
  id: string
  ref: string
  site_id: string
  site_name?: string
  report_id?: string
  report_label?: string
  interval_id?: string
  metric?: DisputeMetric
  period_from?: string
  period_to?: string
  category: DisputeCategory
  title: string
  description: string
  status: DisputeStatus
  waiting_on?: DisputeParty
  outcome?: DisputeOutcome
  resolution?: string
  resolution_report_id?: string
  resolution_report_label?: string
  resolved_at?: string
  acknowledged_at?: string
  reopen_count: number
  created_at: string
  updated_at: string
  raised_by?: string
  raised_by_name?: string
  owner_id?: string
  owner_name?: string
  resolved_by?: string
  resolved_by_name?: string
}

interface DisputeEventDTO {
  id: string
  party: DisputeParty
  kind: DisputeEventKind
  from_status?: DisputeStatus
  to_status?: DisputeStatus
  body?: string
  internal?: boolean
  author_id?: string
  author_name?: string
  created_at: string
}

interface DisputeListResponseDTO {
  disputes?: DisputeDTO[]
  total: number
}

interface DisputeSummaryDTO {
  open: number
  with_coloplus: number
  with_customer: number
}

export function mapDispute(dto: DisputeDTO): Dispute {
  return {
    id: dto.id,
    ref: dto.ref,
    siteId: dto.site_id,
    siteName: dto.site_name,
    reportId: dto.report_id,
    reportLabel: dto.report_label,
    intervalId: dto.interval_id,
    metric: dto.metric,
    periodFrom: dto.period_from,
    periodTo: dto.period_to,
    category: dto.category,
    title: dto.title,
    description: dto.description,
    status: dto.status,
    waitingOn: dto.waiting_on,
    outcome: dto.outcome,
    resolution: dto.resolution,
    resolutionReportId: dto.resolution_report_id,
    resolutionReportLabel: dto.resolution_report_label,
    resolvedAt: dto.resolved_at,
    acknowledgedAt: dto.acknowledged_at,
    reopenCount: dto.reopen_count,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
    raisedBy: dto.raised_by,
    raisedByName: dto.raised_by_name,
    ownerId: dto.owner_id,
    ownerName: dto.owner_name,
    resolvedBy: dto.resolved_by,
    resolvedByName: dto.resolved_by_name,
  }
}

function mapEvent(dto: DisputeEventDTO): DisputeEvent {
  return {
    id: dto.id,
    party: dto.party,
    kind: dto.kind,
    fromStatus: dto.from_status,
    toStatus: dto.to_status,
    body: dto.body ?? '',
    internal: dto.internal,
    authorId: dto.author_id,
    authorName: dto.author_name,
    createdAt: dto.created_at,
  }
}

export interface DisputeFilter {
  siteId?: string
  status?: DisputeStatus
  openOnly?: boolean
}

export async function fetchDisputes(filter: DisputeFilter = {}): Promise<Dispute[]> {
  try {
    const { data } = await getApi().get<DisputeListResponseDTO>('/disputes', {
      params: {
        site_id: filter.siteId,
        status: filter.status,
        open: filter.openOnly ? 'true' : undefined,
        limit: 200,
      },
    })
    return (data.disputes ?? []).map(mapDispute)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load disputes.')
  }
}

export async function fetchDisputeDetail(id: string): Promise<DisputeDetail> {
  try {
    const { data } = await getApi().get<{ dispute: DisputeDTO; events?: DisputeEventDTO[] }>(
      `/disputes/${id}`
    )
    return { dispute: mapDispute(data.dispute), events: (data.events ?? []).map(mapEvent) }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this dispute.')
  }
}

export async function fetchDisputeSummary(): Promise<DisputeSummary> {
  try {
    const { data } = await getApi().get<DisputeSummaryDTO>('/disputes/summary')
    return { open: data.open, withColoplus: data.with_coloplus, withCustomer: data.with_customer }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the dispute summary.')
  }
}

export async function raiseDispute(input: RaiseDisputeInput): Promise<Dispute> {
  try {
    const { data } = await getApi().post<DisputeDTO>(`/sites/${input.siteId}/disputes`, {
      report_id: input.reportId,
      interval_id: input.intervalId,
      metric: input.metric,
      period_from: input.periodFrom,
      period_to: input.periodTo,
      category: input.category,
      title: input.title,
      description: input.description,
    })
    return mapDispute(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not raise this dispute.')
  }
}

export async function commentDispute(
  id: string,
  body: string,
  internal = false
): Promise<DisputeEvent> {
  try {
    const { data } = await getApi().post<DisputeEventDTO>(`/disputes/${id}/comments`, {
      body,
      internal,
    })
    return mapEvent(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not add this comment.')
  }
}

export interface TransitionDisputeInput {
  status: DisputeStatus
  note?: string
  /** Required to resolve: upheld or not_upheld. Upheld needs the correction
      to have been agreed first. */
  outcome?: DisputeOutcome
  /** The corrected, approved report revision that settles an upheld dispute
      about a report. */
  resolutionReportId?: string
}

export async function transitionDispute(
  id: string,
  input: TransitionDisputeInput
): Promise<Dispute> {
  try {
    const { data } = await getApi().post<DisputeDTO>(`/disputes/${id}/transition`, {
      status: input.status,
      note: input.note,
      outcome: input.outcome,
      resolution_report_id: input.resolutionReportId,
    })
    return mapDispute(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not update this dispute.')
  }
}
