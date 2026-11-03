import { getApi } from './api'
import { mapAxiosError } from './auth'
import { sha256Hex } from '../utils'
import type { Attachment, AttachmentOwnerType, AttachmentStatus, UploadTarget } from '../types'

interface AttachmentDTO {
  id: string
  owner_type: AttachmentOwnerType
  owner_id: string
  site_id: string
  filename: string
  content_type: string
  size_bytes: number
  sha256: string
  status: AttachmentStatus
  party: 'airtel' | 'coloplus'
  created_at: string
  confirmed_at?: string
  internal?: boolean
  uploaded_by?: string
}

interface UploadTargetDTO {
  method: string
  url: string
  headers: Record<string, string>
  expires_at: string
}

interface RequestUploadResponseDTO {
  attachment: AttachmentDTO
  upload: UploadTargetDTO
}

interface AttachmentListResponseDTO {
  attachments: AttachmentDTO[]
}

interface DownloadResponseDTO {
  url: string
  expires_at: string
  filename: string
}

function mapAttachment(dto: AttachmentDTO): Attachment {
  return {
    id: dto.id,
    ownerType: dto.owner_type,
    ownerId: dto.owner_id,
    siteId: dto.site_id,
    filename: dto.filename,
    contentType: dto.content_type,
    sizeBytes: dto.size_bytes,
    sha256: dto.sha256,
    status: dto.status,
    party: dto.party,
    createdAt: dto.created_at,
    confirmedAt: dto.confirmed_at,
    internal: dto.internal,
    uploadedBy: dto.uploaded_by,
  }
}

function mapUploadTarget(dto: UploadTargetDTO): UploadTarget {
  return { method: dto.method, url: dto.url, headers: dto.headers, expiresAt: dto.expires_at }
}

export async function fetchAttachments(
  ownerType: AttachmentOwnerType,
  ownerId: string
): Promise<Attachment[]> {
  try {
    const { data } = await getApi().get<AttachmentListResponseDTO>('/attachments', {
      params: { owner_type: ownerType, owner_id: ownerId },
    })
    return data.attachments.map(mapAttachment)
  } catch (error) {
    throw mapAxiosError(error, 'Could not load the files for this record.')
  }
}

interface RequestUploadParams {
  ownerType: AttachmentOwnerType
  ownerId: string
  file: File
  sha256: string
  shared?: boolean
}

async function requestUpload(
  params: RequestUploadParams
): Promise<{ attachment: Attachment; upload: UploadTarget }> {
  try {
    const { data } = await getApi().post<RequestUploadResponseDTO>('/attachments', {
      owner_type: params.ownerType,
      owner_id: params.ownerId,
      filename: params.file.name,
      content_type: params.file.type,
      size_bytes: params.file.size,
      sha256: params.sha256,
      shared: params.shared,
    })
    return { attachment: mapAttachment(data.attachment), upload: mapUploadTarget(data.upload) }
  } catch (error) {
    throw mapAxiosError(error, 'Could not start this upload.')
  }
}

async function putFileToR2(file: File, target: UploadTarget): Promise<void> {
  let response: Response
  try {
    response = await fetch(target.url, {
      method: target.method,
      headers: target.headers,
      body: file,
    })
  } catch {
    throw new Error('Could not reach file storage. Check your connection and try again.')
  }
  if (!response.ok) {
    throw new Error('The upload was rejected by file storage. Try again.')
  }
}

async function completeUpload(id: string): Promise<Attachment> {
  try {
    const { data } = await getApi().post<AttachmentDTO>(`/attachments/${id}/complete`)
    return mapAttachment(data)
  } catch (error) {
    throw mapAxiosError(
      error,
      "The uploaded file didn't match what was described, so it was discarded. Try again."
    )
  }
}

export async function fetchDownloadUrl(id: string): Promise<{ url: string; filename: string }> {
  try {
    const { data } = await getApi().get<DownloadResponseDTO>(`/attachments/${id}/download`)
    return { url: data.url, filename: data.filename }
  } catch (error) {
    throw mapAxiosError(error, 'Could not get a download link for this file.')
  }
}

export interface UploadAttachmentInput {
  ownerType: AttachmentOwnerType
  ownerId: string
  file: File
  shared?: boolean
}

/** The orchestrator the UI calls: hash -> request a link -> PUT -> confirm. */
export async function uploadAndRegisterAttachment(
  input: UploadAttachmentInput
): Promise<Attachment> {
  const digest = await sha256Hex(input.file)
  const { attachment, upload } = await requestUpload({
    ownerType: input.ownerType,
    ownerId: input.ownerId,
    file: input.file,
    sha256: digest,
    shared: input.shared,
  })
  await putFileToR2(input.file, upload)
  return completeUpload(attachment.id)
}
