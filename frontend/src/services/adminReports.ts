import { getApi } from './api'
import { mapAxiosError } from './auth'
import type {
  DailyReport,
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

interface ReportDTO {
  id: string
  site_id: string
  report_date: string
  revision: number
  status: ReportStatus
  author_id: string
  notes: string
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
  limit?: number
  offset?: number
}): Promise<{ reports: DailyReport[]; total: number }> {
  try {
    const { data } = await getApi().get<ReportListResponseDTO>('/daily-reports', {
      params: { status: params.status, limit: params.limit ?? 200, offset: params.offset },
    })
    return { reports: (data.reports ?? []).map(mapDailyReport), total: data.total }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load reports.')
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
