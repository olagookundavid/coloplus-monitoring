import { useState, type FormEvent } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { Button, Field, Input, SegmentedToggle, Select, SidePanel, Spinner, Textarea } from '../ui'
import { useApprovedReportForDate, useReporterIncidents } from '../../hooks'
import type {
  IncidentCorrectionField,
  MySiteRow,
  PowerStatus,
} from '../../types'
import { correctionFieldLabel } from '../../utils'
import type { ReportCorrectionField } from '../../hooks'

export type CorrectionTarget = 'report' | 'incident'

// What the reporter is about to send, handed up to the page so its single
// ConfirmDialog (frontend/CLAUDE.md §2) can name the record and run the
// right mutation once they confirm -- this modal never submits directly.
export type PendingCorrectionRequest =
  | {
      target: 'report'
      reportId: string
      field: ReportCorrectionField
      correctedValue: string
      reason: string
      siteLabel: string
      fieldLabel: string
    }
  | {
      target: 'incident'
      incidentId: string
      field: IncidentCorrectionField
      correctedValue: string
      reason: string
      siteLabel: string
      fieldLabel: string
    }

export interface RequestCorrectionModalInitial {
  incidentId: string
  siteId: string
  field: IncidentCorrectionField
}

export interface RequestCorrectionModalProps {
  open: boolean
  onClose: () => void
  sites: MySiteRow[]
  onRequestConfirm: (pending: PendingCorrectionRequest) => void
  /** Pre-fills the incident target, for "Request again" on a declined request. */
  initial?: RequestCorrectionModalInitial | null
}

const REPORT_FIELD_LABELS: Record<ReportCorrectionField, string> = {
  generator_hours: 'Generator runtime (hours)',
  power_status: 'Grid power today',
}
const REPORT_FIELD_OPTIONS = Object.keys(REPORT_FIELD_LABELS) as ReportCorrectionField[]

const POWER_STATUS_LABELS: Record<PowerStatus, string> = {
  served: 'Power on all day',
  outage: 'Had an outage',
}

const INCIDENT_FIELD_OPTIONS: IncidentCorrectionField[] = ['started_at', 'restored_at']

function todayIsoDate(): string {
  return new Date().toISOString().slice(0, 10)
}

function nowAsDatetimeLocal(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function RequestCorrectionModal({
  open,
  onClose,
  sites,
  onRequestConfirm,
  initial,
}: RequestCorrectionModalProps) {
  const [target, setTarget] = useState<CorrectionTarget>(initial ? 'incident' : 'report')
  const [siteId, setSiteId] = useState(initial?.siteId ?? '')
  const [reportDate, setReportDate] = useState(todayIsoDate())
  const [reportField, setReportField] = useState<ReportCorrectionField>('generator_hours')
  const [incidentId, setIncidentId] = useState(initial?.incidentId ?? '')
  const [incidentField, setIncidentField] = useState<IncidentCorrectionField>(initial?.field ?? 'restored_at')
  const [correctValue, setCorrectValue] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | null>(null)

  const resolvedReportQuery = useApprovedReportForDate(target === 'report' ? siteId : '', reportDate)
  const resolvedReport = resolvedReportQuery.data

  const incidentsQuery = useReporterIncidents()
  const siteIncidents = (incidentsQuery.data ?? []).filter((i) => i.siteId === siteId)

  function resetForm() {
    setTarget(initial ? 'incident' : 'report')
    setSiteId(initial?.siteId ?? '')
    setReportDate(todayIsoDate())
    setReportField('generator_hours')
    setIncidentId(initial?.incidentId ?? '')
    setIncidentField(initial?.field ?? 'restored_at')
    setCorrectValue('')
    setReason('')
    setError(null)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  function changeTarget(next: CorrectionTarget) {
    setTarget(next)
    setCorrectValue('')
    setError(null)
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const site = sites.find((s) => s.siteId === siteId)
    if (!site) {
      setError('Choose which site this is for.')
      return
    }
    if (!reason.trim()) {
      setError('Explain why this needs correcting.')
      return
    }

    if (target === 'report') {
      if (!resolvedReport) {
        setError('No approved report for this site and date yet.')
        return
      }
      if (!correctValue) {
        setError('Enter the correct value.')
        return
      }
      if (reportField === 'generator_hours') {
        const hours = Number(correctValue)
        if (Number.isNaN(hours) || hours < 0 || hours > 24) {
          setError('Generator runtime hours must be between 0 and 24.')
          return
        }
      }
      setError(null)
      onRequestConfirm({
        target: 'report',
        reportId: resolvedReport.id,
        field: reportField,
        correctedValue: correctValue,
        reason: reason.trim(),
        siteLabel: `${site.airtelCode} · ${site.name}`,
        fieldLabel: REPORT_FIELD_LABELS[reportField],
      })
      handleClose()
      return
    }

    if (!incidentId) {
      setError('Choose which incident this is for.')
      return
    }
    if (!correctValue) {
      setError('Enter the correct time.')
      return
    }
    const correctedIso = new Date(correctValue).toISOString()
    if (new Date(correctedIso).getTime() > Date.now()) {
      setError('The corrected time cannot be in the future.')
      return
    }
    setError(null)
    onRequestConfirm({
      target: 'incident',
      incidentId,
      field: incidentField,
      correctedValue: correctedIso,
      reason: reason.trim(),
      siteLabel: `${site.airtelCode} · ${site.name}`,
      fieldLabel: correctionFieldLabel(incidentField),
    })
    handleClose()
  }

  return (
    <SidePanel
      open={open}
      onClose={handleClose}
      eyebrow="Corrections"
      title="Request a correction"
      subtitle="For records already submitted. Your supervisor reviews the request and the original stays on file."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="request-correction-form">
            Send request
          </Button>
        </>
      }
    >
      <form
        id="request-correction-form"
        noValidate
        onSubmit={handleSubmit}
        className="flex flex-col gap-4"
      >
        {error && (
          <p className="inline-flex items-center gap-1.5 text-sm text-danger">
            <ExclamationTriangleIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
            {error}
          </p>
        )}

        {!initial && (
          <SegmentedToggle<CorrectionTarget>
            aria-label="What this correction is for"
            value={target}
            onChange={changeTarget}
            options={[
              { value: 'report', label: 'Daily report' },
              { value: 'incident', label: 'Incident' },
            ]}
          />
        )}

        <Field label="Site" htmlFor="correction-site">
          <Select
            id="correction-site"
            value={siteId}
            onChange={(event) => setSiteId(event.target.value)}
            disabled={Boolean(initial)}
          >
            <option value="">Select a site</option>
            {sites.map((site) => (
              <option key={site.siteId} value={site.siteId}>
                {site.airtelCode} · {site.name}
              </option>
            ))}
          </Select>
        </Field>

        {target === 'report' ? (
          <>
            <Field label="Report date" htmlFor="correction-report-date">
              <Input
                id="correction-report-date"
                type="date"
                max={todayIsoDate()}
                value={reportDate}
                onChange={(event) => setReportDate(event.target.value)}
              />
            </Field>
            {siteId && reportDate && (
              <p className="text-xs text-text-muted">
                {resolvedReportQuery.isLoading ? (
                  <Spinner className="h-4 w-4" />
                ) : resolvedReport ? (
                  `Approved report found, revision ${resolvedReport.revision}.`
                ) : (
                  'No approved report for this site and date yet.'
                )}
              </p>
            )}

            <Field label="Field" htmlFor="correction-report-field">
              <Select
                id="correction-report-field"
                value={reportField}
                onChange={(event) => {
                  setReportField(event.target.value as ReportCorrectionField)
                  setCorrectValue('')
                }}
              >
                {REPORT_FIELD_OPTIONS.map((f) => (
                  <option key={f} value={f}>
                    {REPORT_FIELD_LABELS[f]}
                  </option>
                ))}
              </Select>
            </Field>

            <Field label="Correct value" htmlFor="correction-report-value">
              {reportField === 'generator_hours' ? (
                <Input
                  id="correction-report-value"
                  type="number"
                  min={0}
                  max={24}
                  step={0.5}
                  inputMode="decimal"
                  placeholder="e.g. 4.5"
                  value={correctValue}
                  onChange={(event) => setCorrectValue(event.target.value)}
                />
              ) : (
                <Select
                  id="correction-report-value"
                  value={correctValue}
                  onChange={(event) => setCorrectValue(event.target.value)}
                >
                  <option value="">Select</option>
                  {(Object.keys(POWER_STATUS_LABELS) as PowerStatus[]).map((status) => (
                    <option key={status} value={status}>
                      {POWER_STATUS_LABELS[status]}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
          </>
        ) : (
          <>
            <Field label="Incident" htmlFor="correction-incident">
              <Select
                id="correction-incident"
                value={incidentId}
                onChange={(event) => setIncidentId(event.target.value)}
                disabled={Boolean(initial)}
              >
                <option value="">Select an incident</option>
                {siteIncidents.map((incident) => (
                  <option key={incident.id} value={incident.id}>
                    {incident.ref} · {incident.title}
                  </option>
                ))}
              </Select>
            </Field>

            <Field label="Field" htmlFor="correction-incident-field">
              <Select
                id="correction-incident-field"
                value={incidentField}
                onChange={(event) => setIncidentField(event.target.value as IncidentCorrectionField)}
                disabled={Boolean(initial)}
              >
                {INCIDENT_FIELD_OPTIONS.map((f) => (
                  <option key={f} value={f}>
                    {correctionFieldLabel(f)}
                  </option>
                ))}
              </Select>
            </Field>

            <Field label="Correct value" htmlFor="correction-incident-value">
              <Input
                id="correction-incident-value"
                type="datetime-local"
                max={nowAsDatetimeLocal()}
                value={correctValue}
                onChange={(event) => setCorrectValue(event.target.value)}
              />
            </Field>
          </>
        )}

        <Field label="Reason" htmlFor="correction-reason">
          <Textarea
            id="correction-reason"
            placeholder="Why does this need correcting?"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </Field>
      </form>
    </SidePanel>
  )
}
