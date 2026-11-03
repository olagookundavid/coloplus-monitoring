// Mirrors dto.AttachmentDTO/UploadTarget/RequestUploadRequest
// (backend/internal/dto/attachments.go). Backend only supports these four
// owner types (domain.ValidAttachmentOwner) -- never a daily report, never
// a correction.
export type AttachmentOwnerType = 'ticket' | 'incident' | 'dispute' | 'acceptance'

export type AttachmentStatus = 'pending' | 'ready' | 'rejected' | 'expired'
export type AttachmentParty = 'airtel' | 'coloplus'

export interface Attachment {
  id: string
  ownerType: AttachmentOwnerType
  ownerId: string
  siteId: string
  filename: string
  contentType: string
  sizeBytes: number
  sha256: string
  status: AttachmentStatus
  party: AttachmentParty
  createdAt: string
  confirmedAt?: string
  /** Staff-only; absent for a customer (backend strips it). */
  internal?: boolean
  /** Staff-only; absent for a customer. */
  uploadedBy?: string
}

// The upload target returned alongside a pending attachment -- a presigned,
// cross-origin PUT straight to R2, not through the Coloplus API.
export interface UploadTarget {
  method: string
  url: string
  headers: Record<string, string>
  expiresAt: string
}

export interface RequestUploadInput {
  ownerType: AttachmentOwnerType
  ownerId: string
  filename: string
  contentType: string
  sizeBytes: number
  sha256: string
  /** Coloplus-only; a Coloplus upload defaults hidden from Airtel unless set. */
  shared?: boolean
}
