import type { IssuedExport, IssuedExportDownload, IssuedExportListResult } from '../types'
import { getApi } from './api'
import { mapAxiosError } from './auth'

interface IssuedExportDTO {
  id: string
  kind: string
  filename: string
  params: Record<string, unknown>
  row_count: number
  size_bytes: number
  sha256: string
  stored: boolean
  issued_by: string
  created_at: string
}

interface IssuedExportListResponseDTO {
  exports: IssuedExportDTO[]
  total: number
  limit: number
  offset: number
}

interface DownloadResponseDTO {
  url: string
  expires_at: string
  filename: string
}

function mapIssuedExport(dto: IssuedExportDTO): IssuedExport {
  return {
    id: dto.id,
    kind: dto.kind,
    filename: dto.filename,
    params: dto.params,
    rowCount: dto.row_count,
    sizeBytes: dto.size_bytes,
    sha256: dto.sha256,
    stored: dto.stored,
    issuedBy: dto.issued_by,
    createdAt: dto.created_at,
  }
}

export interface IssuedExportsQuery {
  limit?: number
  offset?: number
}

// Who sees what is entirely server-side: audit.read sees every export,
// exports.create alone sees only the caller's own. There's no filter
// param to pass -- the response is already scoped to the actor.
export async function fetchIssuedExports(
  query: IssuedExportsQuery = {}
): Promise<IssuedExportListResult> {
  try {
    const { data } = await getApi().get<IssuedExportListResponseDTO>('/exports/issued', {
      params: query,
    })
    return {
      exports: data.exports.map(mapIssuedExport),
      total: data.total,
      limit: data.limit,
      offset: data.offset,
    }
  } catch (error) {
    throw mapAxiosError(error, 'Could not load issued exports.')
  }
}

export async function fetchIssuedExportDownload(id: string): Promise<IssuedExportDownload> {
  try {
    const { data } = await getApi().get<DownloadResponseDTO>(`/exports/issued/${id}/download`)
    return { url: data.url, expiresAt: data.expires_at, filename: data.filename }
  } catch (error) {
    throw mapAxiosError(error, 'Could not get a download link for this export.')
  }
}
