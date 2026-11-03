// Mirrors the exact allow-list in backend/internal/domain/attachment.go --
// content type -> the filename extension(s) it must match. Checked
// client-side before a file is ever hashed or sent, so a bad file never
// reaches the network.
const ALLOWED_TYPES: Record<string, string[]> = {
  'image/jpeg': ['.jpg', '.jpeg'],
  'image/png': ['.png'],
  'image/webp': ['.webp'],
  'image/heic': ['.heic', '.heif'],
  'application/pdf': ['.pdf'],
  'text/plain': ['.txt', '.log'],
  'text/csv': ['.csv'],
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx'],
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': ['.docx'],
}

export const ACCEPT_ATTR = Object.values(ALLOWED_TYPES).flat().join(',')

export const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024

function hasExtension(filename: string, extensions: string[]): boolean {
  const lower = filename.toLowerCase()
  return extensions.some((ext) => lower.endsWith(ext))
}

function formatMB(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/** A client-side pre-check mirroring the backend's own validation, so the
    error shows up immediately rather than after a network round trip. */
export function validateAttachmentFile(file: File): string | undefined {
  if (file.size <= 0) return 'This file is empty.'
  if (file.size > MAX_ATTACHMENT_BYTES) {
    return `This file is too large (${formatMB(file.size)}). The limit is ${formatMB(MAX_ATTACHMENT_BYTES)}.`
  }
  const extensions = ALLOWED_TYPES[file.type]
  if (!extensions) {
    return 'That file type is not accepted. Use a photo, PDF, text, CSV, Word or Excel file.'
  }
  if (!hasExtension(file.name, extensions)) {
    return "This file's name doesn't match its type."
  }
  return undefined
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return formatMB(bytes)
}

/** file.arrayBuffer() -> crypto.subtle.digest -> lowercase hex, as the
    presigned upload link's signature pins this value. */
export async function sha256Hex(file: File): Promise<string> {
  const buffer = await file.arrayBuffer()
  const digest = await crypto.subtle.digest('SHA-256', buffer)
  return Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('')
}
