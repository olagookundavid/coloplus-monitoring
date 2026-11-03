// The in-app alert centre row and the weekly-digest preference -- mirrors
// dto.AlertDTO / dto.AlertListResponse / dto.DigestPreferenceDTO
// (backend/internal/dto/alerts.go). Unlike a Notification, an Alert's body
// is returned in full (it's the same text as the email).

export interface Alert {
  id: string
  /** Not shown directly -- title/body already read as plain sentences. */
  kind: string
  title: string
  body: string
  createdAt: string
  acknowledgedAt?: string
}

export interface AlertListResult {
  alerts: Alert[]
  total: number
  /** The exact count of all open (unacknowledged) alerts, independent of this call's own `unread`/`limit`/`offset`. */
  unread: number
  limit: number
  offset: number
}

export interface DigestPreference {
  weekly: boolean
}
