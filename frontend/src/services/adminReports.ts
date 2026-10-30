import { getApi } from './api'
import { mapAxiosError } from './auth'

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
  author_id: string
  supersedes_id?: string
  submitted_at?: string
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
