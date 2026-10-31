import type { AuditCategory, AuditEvent, ManagedUser } from '../types'

// audit_log has no category column -- this buckets by action/metadata.
// user.updated whose `changed` touched site_count is checked first, since
// it's the one deterministic "this was a site assignment" signal
// (backend/internal/services/users.go); everything else `user.*`/`auth.*`
// falls under User. Shared by the Audit log page and the notifications feed
// so both describe the same event the same way.
export function categorizeAuditEvent(event: AuditEvent): AuditCategory {
  const changed = event.metadata.changed
  if (
    event.action === 'user.updated' &&
    typeof changed === 'object' &&
    changed !== null &&
    'site_count' in changed
  ) {
    return 'assignment'
  }
  if (event.action.startsWith('user.') || event.action.startsWith('auth.')) return 'user'
  if (
    event.action === 'report.submitted' ||
    event.action === 'report.approved' ||
    event.action === 'report.rejected' ||
    event.action === 'report.correction_started'
  ) {
    return 'approval'
  }
  return 'other'
}

const ACTION_LABELS: Record<string, string> = {
  'user.created': 'Created a user account',
  'user.updated': 'Updated a user account',
  'user.invite_resent': 'Resent an invite',
  'user.initial_admin_created': 'Created the initial administrator account',
  'auth.login': 'Signed in',
  'auth.login_failed': 'Had a failed sign-in attempt',
  'auth.mfa_enabled': 'Turned on two-step verification',
  'auth.mfa_reset': 'Reset two-step verification',
  'auth.recovery_code_used': 'Used a recovery code',
  'auth.recovery_codes_renewed': 'Regenerated recovery codes',
  'auth.reset_requested': 'Requested a password reset',
  'auth.password_set': 'Set a new password',
  'report.submitted': 'Submitted a daily report for approval',
  'report.approved': 'Approved a daily report',
  'report.rejected': 'Returned a daily report for correction',
  'report.correction_started': 'Started a report correction',
  'site.created': 'Created a site',
  'site.updated': 'Updated a site',
  'site.activated': 'Activated a site',
  'incident.reported': 'Reported an incident',
  'incident.assigned': 'Assigned an incident',
  'incident.updated': 'Updated an incident',
  'incident.published': 'Published an incident',
  'dispute.raised': 'Raised a dispute',
  'acceptance.requested': 'Requested site acceptance',
  'export.daily_summaries': 'Exported daily summaries',
  'notification.failed': 'A notification failed to send',
  'notification.retried': 'Retried a notification',
  'system.started': 'The system started',
  'system.stopping': 'The system began shutting down',
}

function humanizeAction(action: string): string {
  const words = action.replace(/[._]/g, ' ')
  return words.charAt(0).toUpperCase() + words.slice(1)
}

// Only names the entity when it's a *different* user from the actor and we
// can actually resolve a name -- otherwise just the plain action phrase, no
// invented specifics beyond what the record carries.
export function describeAuditEvent(
  event: AuditEvent,
  category: AuditCategory,
  usersById: Map<string, ManagedUser>
): string {
  const base =
    category === 'assignment'
      ? 'Updated site assignment'
      : (ACTION_LABELS[event.action] ?? humanizeAction(event.action))
  if (event.entityType === 'user' && event.entityId && event.entityId !== event.actorId) {
    const targetName = usersById.get(event.entityId)?.name
    if (targetName) return `${base} for ${targetName}`
  }
  return base
}

export function resolveAuditActorName(
  event: AuditEvent,
  usersById: Map<string, ManagedUser>
): string {
  if (!event.actorId) return 'System'
  return usersById.get(event.actorId)?.name ?? event.actorId
}
