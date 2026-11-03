// The admin-facing email-delivery log row -- mirrors dto.NotificationDTO /
// dto.NotificationListResponse (backend/internal/dto/auth.go). The message
// text is deliberately never returned by the API (it can hold a one-time
// link), so there is no `body` field here at all.

export type NotificationStatus = 'pending' | 'sent' | 'failed'

export interface Notification {
  id: string
  /** Dot-separated, e.g. "report.anomaly" -- not a closed set. */
  kind: string
  recipient: string
  subject: string
  status: NotificationStatus
  attempts: number
  lastError?: string
  createdAt: string
  sentAt?: string
}

export interface NotificationListResult {
  notifications: Notification[]
  total: number
  limit: number
  offset: number
}
