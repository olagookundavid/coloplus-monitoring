import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  DailyReport,
  PowerStatus,
  ReportCalculation,
  ReportDetail,
  ReportStatus,
  ReportSummary,
} from '../types'

// "Correction" in the Overview approvals table isn't a separate backend
// workflow -- it's the same submitted-report queue, filtered by
// supersedes_id != "" (backend/internal/services/reports.go review()).
export interface PendingApproval {
  id: string
  siteId: string
  reportDate: string
  authorId: string
  isCorrection: boolean
  submittedAt?: string
}

interface IntervalDTO {
  start: string
  end: string
  state: string
  source?: string
  basis?: string
}

interface ReportDTO {
  id: string
  site_id: string
  report_date: string
  revision: number
  status: ReportStatus
  author_id: string
  notes: string
  intervals?: IntervalDTO[]
  generator_hours?: number | null
  outage_reason?: string
  submitted_at?: string
  reviewed_at?: string
  reviewer_id?: string
  review_comment?: string
  supersedes_id?: string
  correction_reason?: string
  row_version: number
  created_at: string
  updated_at: string
}

interface SummaryDTO {
  eligible_minutes: number
  served_minutes: number
  outage_minutes: number
  unknown_minutes: number
  coverage_pct: number | null
  reported_availability_pct: number | null
  confirmed_served_share_pct: number | null
  provisional: boolean
}

interface CalculationDTO {
  formula: string
  version: string
  basis: string
  input_report_id: string
  input_revision: number
  window_start: string
  window_end: string
  timezone: string
  computed_at: string
  explanation: string
  warnings: string[]
}

interface ReportDetailResponseDTO {
  report?: ReportDTO
  summary: SummaryDTO
  calculation: CalculationDTO
  open_disputes: number
}

interface ReportListResponseDTO {
  reports?: ReportDTO[]
  total: number
}

function mapPendingApproval(dto: ReportDTO): PendingApproval {
  return {
    id: dto.id,
    siteId: dto.site_id,
    reportDate: dto.report_date,
    authorId: dto.author_id,
    isCorrection: Boolean(dto.supersedes_id),
    submittedAt: dto.submitted_at,
  }
}

function mapDailyReport(dto: ReportDTO): DailyReport {
  return {
    id: dto.id,
    siteId: dto.site_id,
    reportDate: dto.report_date,
    revision: dto.revision,
    status: dto.status,
    authorId: dto.author_id,
    notes: dto.notes,
    intervals: dto.intervals?.map((iv) => ({
      start: iv.start,
      end: iv.end,
      state: iv.state as PowerStatus,
      source: iv.source,
      basis: iv.basis as 'observed' | 'declared' | undefined,
    })),
    generatorHours: dto.generator_hours ?? null,
    outageReason: dto.outage_reason,
    submittedAt: dto.submitted_at,
    reviewedAt: dto.reviewed_at,
    reviewerId: dto.reviewer_id,
    reviewComment: dto.review_comment,
    supersedesId: dto.supersedes_id,
    correctionReason: dto.correction_reason,
    rowVersion: dto.row_version,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

function mapSummary(dto: SummaryDTO): ReportSummary {
  return {
    eligibleMinutes: dto.eligible_minutes,
    servedMinutes: dto.served_minutes,
    outageMinutes: dto.outage_minutes,
    unknownMinutes: dto.unknown_minutes,
    coveragePct: dto.coverage_pct,
    reportedAvailabilityPct: dto.reported_availability_pct,
    confirmedServedSharePct: dto.confirmed_served_share_pct,
    provisional: dto.provisional,
  }
}

function mapCalculation(dto: CalculationDTO): ReportCalculation {
  return {
    formula: dto.formula,
    version: dto.version,
    basis: dto.basis,
    inputReportId: dto.input_report_id,
    inputRevision: dto.input_revision,
    windowStart: dto.window_start,
    windowEnd: dto.window_end,
    timezone: dto.timezone,
    computedAt: dto.computed_at,
    explanation: dto.explanation,
    warnings: dto.warnings ?? [],
  }
}

export async function fetchPendingApprovals(): Promise<PendingApproval[]> {
  try {
    const { data } = await getApi().get<{ reports?: ReportDTO[] }>('/daily-reports', {
      params: { status: 'submitted', limit: 200 },
    })
    return (data.reports ?? []).map(mapPendingApproval)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load pending approvals.')
  }
}

export async function fetchReports(params: {
  status?: ReportStatus
  mine?: boolean
  siteId?: string
  /** yyyy-mm-dd, inclusive. */
  from?: string
  to?: string
  limit?: number
  offset?: number
}): Promise<{ reports: DailyReport[]; total: number }> {
  try {
    const { data } = await getApi().get<ReportListResponseDTO>('/daily-reports', {
      params: {
        status: params.status,
        mine: params.mine ? 'true' : undefined,
        site_id: params.siteId,
        from: params.from,
        to: params.to,
        limit: params.limit ?? 200,
        offset: params.offset,
      },
    })
    return { reports: (data.reports ?? []).map(mapDailyReport), total: data.total }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load reports.')
  }
}

// The reporter's simplified day-level entry: a power-status toggle and an
// optional generator-hours number, no hand-entered intervals. The backend
// builds the day's single interval from powerStatus itself.
export interface DailyReportFormInput {
  powerStatus: PowerStatus
  generatorHours: number | null
  outageReason: string
  notes: string
}

function reportFormBody(input: DailyReportFormInput) {
  return {
    power_status: input.powerStatus,
    generator_hours: input.generatorHours,
    outage_reason: input.outageReason,
    notes: input.notes,
  }
}

export async function createDailyReport(
  siteId: string,
  reportDate: string,
  input: DailyReportFormInput
): Promise<DailyReport> {
  try {
    const { data } = await getApi().post<ReportDTO>(`/sites/${siteId}/daily-reports`, {
      report_date: reportDate,
      ...reportFormBody(input),
    })
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not start this report.')
  }
}

export async function saveDailyReportDraft(
  id: string,
  rowVersion: number,
  input: DailyReportFormInput
): Promise<DailyReport> {
  try {
    const { data } = await getApi().put<ReportDTO>(`/daily-reports/${id}`, {
      row_version: rowVersion,
      ...reportFormBody(input),
    })
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not save this report.')
  }
}

export async function submitDailyReport(id: string): Promise<DailyReport> {
  try {
    // A body is required even though the backend's own fields here are optional:
    // axios only sets Content-Type: application/json when it has an object to
    // stringify, and the API's CSRF guard rejects writes without that header.
    const { data } = await getApi().post<ReportDTO>(`/daily-reports/${id}/submit`, {})
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not submit this report.')
  }
}

// Opens a new, editable revision of an already-approved report, linked via
// supersedes_id -- the original stays visible and unchanged until this
// revision is itself submitted and approved (backend/internal/services
// /reports.go StartCorrection). The reporter then edits it through the same
// save/submit path as any other draft.
export async function startReportCorrection(
  reportId: string,
  reason: string
): Promise<DailyReport> {
  try {
    const { data } = await getApi().post<ReportDTO>(`/daily-reports/${reportId}/corrections`, {
      reason,
    })
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not start this correction.')
  }
}

export async function fetchReportDetail(id: string): Promise<ReportDetail> {
  try {
    const { data } = await getApi().get<ReportDetailResponseDTO>(`/daily-reports/${id}`)
    if (!data.report) {
      throw new Error('This record is not available to Coloplus staff.')
    }
    return {
      report: mapDailyReport(data.report),
      summary: mapSummary(data.summary),
      calculation: mapCalculation(data.calculation),
      openDisputes: data.open_disputes,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load this report.')
  }
}

export async function approveReport(id: string, comment?: string): Promise<DailyReport> {
  try {
    const { data } = await getApi().post<ReportDTO>(`/daily-reports/${id}/approve`, {
      comment: comment ?? '',
    })
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not approve this record.')
  }
}

export async function rejectReport(id: string, reason: string): Promise<DailyReport> {
  try {
    const { data } = await getApi().post<ReportDTO>(`/daily-reports/${id}/reject`, { reason })
    return mapDailyReport(data)
  } catch (error) {
    throw mapAxiosError(error, 'Could not return this record.')
  }
}
