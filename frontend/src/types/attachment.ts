// Mirrors dto.AttachmentDTO/UploadTarget/RequestUploadRequest
// (backend/internal/dto/attachments.go). Backend supports these five owner
// types (domain.ValidAttachmentOwner) -- never a correction, which has no
// id of its own (it's a report revision). `report` evidence is always
// internal (domain/services/attachments.go's `coloplusOnly`) and closed
// once the report is approved or superseded (`frozen`, a 409).
export type AttachmentOwnerType = 'ticket' | 'incident' | 'dispute' | 'acceptance' | 'report'

export type AttachmentStatus = 'pending' | 'ready' | 'rejected' | 'expired'
export type AttachmentParty = 'customer' | 'coloplus'

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
  uploadedByName?: string
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
  /** Coloplus-only; a Coloplus upload defaults hidden from the customer unless set. */
  shared?: boolean
}
