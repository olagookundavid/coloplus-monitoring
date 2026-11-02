import {
  AUDIT_CATEGORY_LABELS,
  AUDIT_CATEGORY_TONES,
  type AuditCategory,
  type AuditEvent,
} from '../../types'
import { formatDateTimeWAT } from '../../utils'
import { SidePanel, StatusBadge } from '../ui'

export interface AuditEventDetailPanelProps {
  event: AuditEvent | null
  description: string
  category: AuditCategory
  actorName: string
  open: boolean
  onClose: () => void
}

// The only audited action carrying anything resembling a before/after diff
// (backend/internal/services/reports.go review(), approving a correction) --
// three summary totals, not a general field diff. Everything else's
// metadata is rendered as plain field/value facts, not labelled as a diff.
const SUMMARY_FIELD_LABELS: Record<string, string> = {
  served_minutes: 'Served minutes',
  outage_minutes: 'Outage minutes',
  unknown_minutes: 'Unknown minutes',
}

function isCorrectionDiff(event: AuditEvent): event is AuditEvent & {
  metadata: { before: Record<string, unknown>; after: Record<string, unknown> }
} {
  return (
    (event.action === 'report.approved' || event.action === 'report.rejected') &&
    typeof event.metadata.before === 'object' &&
    event.metadata.before !== null &&
    typeof event.metadata.after === 'object' &&
    event.metadata.after !== null
  )
}

function formatMetadataValue(value: unknown): string {
  if (value === null || value === undefined) return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function AuditEventDetailPanel({
  event,
  description,
  category,
  actorName,
  open,
  onClose,
}: AuditEventDetailPanelProps) {
  if (!event) return null

  const metadataEntries = Object.entries(event.metadata).filter(
    ([key]) => key !== 'before' && key !== 'after'
  )
  const hasDiff = isCorrectionDiff(event)

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={event.entityType ? `${event.entityType} · ${event.entityId}` : undefined}
      title={description}
      subtitle={
        <div className="flex flex-wrap items-center gap-3">
          <StatusBadge
            label={AUDIT_CATEGORY_LABELS[category]}
            tone={AUDIT_CATEGORY_TONES[category]}
          />
          <span>
            {actorName} · {formatDateTimeWAT(event.occurredAt)}
          </span>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
          <h3 className="text-base font-semibold text-text">Event</h3>
          <dl className="mt-4 divide-y divide-border">
            <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
              <dt className="text-sm text-text-muted">Action</dt>
              <dd className="text-sm font-medium text-text sm:text-right">{event.action}</dd>
            </div>
            <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
              <dt className="text-sm text-text-muted">Entity</dt>
              <dd className="text-sm font-medium text-text sm:text-right">
                {event.entityType || '—'} {event.entityId && `· ${event.entityId}`}
              </dd>
            </div>
          </dl>
        </div>

        {hasDiff && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">What changed</h3>
            <dl className="mt-4 divide-y divide-border">
              {Object.keys(SUMMARY_FIELD_LABELS).map((key) => (
                <div
                  key={key}
                  className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <dt className="text-sm text-text-muted">{SUMMARY_FIELD_LABELS[key]}</dt>
                  <dd className="flex items-center gap-2 text-sm sm:text-right">
                    <span className="text-text-muted line-through">
                      {formatMetadataValue((event.metadata.before as Record<string, unknown>)[key])}
                    </span>
                    <span className="font-medium text-text">
                      {formatMetadataValue((event.metadata.after as Record<string, unknown>)[key])}
                    </span>
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        )}

        {!hasDiff && metadataEntries.length > 0 && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Details</h3>
            <dl className="mt-4 divide-y divide-border">
              {metadataEntries.map(([key, value]) => (
                <div
                  key={key}
                  className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <dt className="text-sm text-text-muted">{key}</dt>
                  <dd className="text-sm font-medium text-text sm:text-right">
                    {formatMetadataValue(value)}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        )}
      </div>
    </SidePanel>
  )
}
