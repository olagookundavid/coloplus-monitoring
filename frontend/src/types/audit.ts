// Mirrors backend/internal/dto.AuditEventDTO. Richer than the AuditEvent
// already exported from services/adminAudit.ts (which only covers Overview's
// small "recent activity" widget) -- this one carries metadata, needed for
// the Audit log page's detail panel.
export interface AuditEvent {
  id: number
  occurredAt: string
  actorId: string
  action: string
  entityType: string
  entityId: string
  metadata: Record<string, unknown>
}

// A client-side classification of `action`/`entityType` -- audit_log has no
// category column. See categorize() in pages/ops/AuditLog.tsx.
export type AuditCategory = 'user' | 'approval' | 'assignment' | 'other'

export const AUDIT_CATEGORY_LABELS: Record<AuditCategory, string> = {
  user: 'User',
  assignment: 'Assignment',
  approval: 'Approval',
  other: 'Other',
}

// Reuses StatusBadge's existing tones rather than inventing a 6-color
// system for 4 categories.
export const AUDIT_CATEGORY_TONES: Record<AuditCategory, 'success' | 'info' | 'neutral'> = {
  approval: 'success',
  user: 'info',
  assignment: 'info',
  other: 'neutral',
}
