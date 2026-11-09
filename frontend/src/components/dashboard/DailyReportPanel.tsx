import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import {
  AlertBanner,
  AttachmentList,
  Button,
  Field,
  FileDropZone,
  Input,
  SegmentedToggle,
  SidePanel,
  Textarea,
} from '../ui'
import { useAttachments, useDailyReportForm } from '../../hooks'
import type { PowerStatus, SiteReportRow } from '../../types'

export type DailyReportPanelMode = 'create' | 'edit' | 'view'

export interface DailyReportPanelProps {
  open: boolean
  onClose: () => void
  site: SiteReportRow
  /** The reporting date, yyyy-mm-dd. */
  reportDate: string
  /** Present for Continue/Fix/View; absent for a fresh Report. */
  reportId?: string
  mode: DailyReportPanelMode
}

function formatReportDateLabel(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1)).toLocaleDateString('en-GB', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  })
}

export function DailyReportPanel({
  open,
  onClose,
  site,
  reportDate,
  reportId,
  mode,
}: DailyReportPanelProps) {
  const {
    detailQuery,
    report,
    powerStatus,
    setPowerStatus,
    generatorHours,
    setGeneratorHours,
    outageReason,
    setOutageReason,
    notes,
    setNotes,
    error,
    submitting,
    readOnly,
    needsLateReason,
    lateReason,
    setLateReason,
    handleSaveDraft,
    handleSubmit,
  } = useDailyReportForm({ site, reportDate, reportId, mode, onSuccess: onClose })

  const returned = report?.status === 'rejected'
  const frozen = report?.status === 'approved' || report?.status === 'superseded'
  const canAttachEvidence = Boolean(report?.id) && !readOnly && !frozen
  const attachmentsQuery = useAttachments('report', report?.id)

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={`Daily report · ${formatReportDateLabel(reportDate)}`}
      title={`${site.customerSiteCode} · ${site.name}`}
      footer={
        readOnly ? (
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        ) : (
          <>
            <Button
              variant="secondary"
              onClick={handleSaveDraft}
              loading={submitting === 'draft'}
              disabled={submitting !== null}
            >
              Save draft
            </Button>
            <Button
              onClick={handleSubmit}
              loading={submitting === 'submit'}
              disabled={submitting !== null}
            >
              Submit report
            </Button>
          </>
        )
      }
    >
      {detailQuery.isLoading ? (
        <p className="text-sm text-text-muted">Loading...</p>
      ) : (
        <div className="flex flex-col gap-5">
          {returned && (
            <AlertBanner
              tone="warning"
              icon={ExclamationTriangleIcon}
              title="Returned for correction"
              description={report?.reviewComment || 'A reason was not given.'}
            />
          )}

          {error && <p className="text-sm text-danger">{error}</p>}

          <Field label="Grid power today">
            <SegmentedToggle<PowerStatus>
              aria-label="Grid power today"
              value={powerStatus}
              onChange={setPowerStatus}
              disabled={readOnly}
              options={[
                { value: 'served', label: 'Power on all day' },
                { value: 'outage', label: 'Had an outage' },
              ]}
            />
          </Field>

          {powerStatus === 'outage' && (
            <Field
              label="Reason for outage"
              htmlFor="outage-reason"
              hint="Required to submit a day with an outage."
            >
              <Textarea
                id="outage-reason"
                value={outageReason}
                onChange={(event) => setOutageReason(event.target.value)}
                disabled={readOnly}
              />
            </Field>
          )}

          <Field
            label="Generator runtime (hours)"
            htmlFor="generator-hours"
            hint="Fuel use is estimated from this and shown to the customer as an estimate."
          >
            <Input
              id="generator-hours"
              type="number"
              min={0}
              max={24}
              step={0.5}
              inputMode="decimal"
              value={generatorHours}
              onChange={(event) => setGeneratorHours(event.target.value)}
              disabled={readOnly}
            />
          </Field>

          <Field label="Notes" htmlFor="report-notes" hint="(optional)">
            <Textarea
              id="report-notes"
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              disabled={readOnly}
            />
          </Field>

          {needsLateReason && (
            <Field
              label="Reason for late submission"
              htmlFor="late-reason"
              hint="This report is later than your site's freshness window allows."
            >
              <Textarea
                id="late-reason"
                value={lateReason}
                onChange={(event) => setLateReason(event.target.value)}
                disabled={readOnly}
              />
            </Field>
          )}

          <Field label="Evidence" hint="A meter photo or similar. Internal to Coloplus only.">
            {report?.id ? (
              <div className="flex flex-col gap-3">
                <AttachmentList
                  attachments={attachmentsQuery.data ?? []}
                  loading={attachmentsQuery.isLoading}
                  emptyMessage="No evidence attached yet."
                />
                {canAttachEvidence && <FileDropZone ownerType="report" ownerId={report.id} isStaff />}
                {frozen && (
                  <p className="text-xs text-text-muted">
                    This report is {report.status}; no more evidence can be added.
                  </p>
                )}
              </div>
            ) : (
              <p className="text-sm text-text-muted">Save a draft first to attach evidence.</p>
            )}
          </Field>
        </div>
      )}
    </SidePanel>
  )
}
