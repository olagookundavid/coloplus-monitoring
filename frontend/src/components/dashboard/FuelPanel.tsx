import { useState, type FormEvent } from 'react'
import { ExclamationTriangleIcon, TrashIcon } from '@heroicons/react/24/outline'

import { SkeletonListItems } from '../skeletons'
import {
  AlertBanner,
  BasisBadge,
  Button,
  CalculationDetails,
  ConfirmDialog,
  DataTable,
  Field,
  Input,
  RowActions,
  Select,
  SidePanel,
  StatusBadge,
  Textarea,
  type CalculationDetailRow,
  type Column,
  type RowAction,
} from '../ui'
import {
  useFuelStatus,
  usePermissions,
  useRecordFuelMutation,
  useVoidFuelMutation,
} from '../../hooks'
import type { FuelEntry, FuelEntryKind } from '../../types'
import {
  formatDateTimeWAT,
  getFieldErrors,
  nameOrId,
  recordFuelSchema,
  toast,
  voidReasonError,
  type RecordFuelFormValues,
} from '../../utils'

export interface FuelPanelProps {
  open: boolean
  onClose: () => void
  siteId: string
  siteName: string
}

// datetime-local works in the browser's local wall-clock time, not UTC --
// this gives "now" in that same format so the field defaults sensibly and
// `max` can block a future pick at the browser level (Joi blocks it again
// on submit, since the browser constraint isn't trustworthy alone).
function nowLocalDatetimeValue(): string {
  const now = new Date()
  const offsetMs = now.getTimezoneOffset() * 60000
  return new Date(now.getTime() - offsetMs).toISOString().slice(0, 16)
}

function initialValues(): RecordFuelFormValues {
  return { kind: 'delivery', litres: '', occurredAt: nowLocalDatetimeValue(), note: '' }
}

// A fresh key per form open, reused across a retried submission of that
// same entry -- a double tap must not record it twice.
function newClientKey(): string | undefined {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : undefined
}

const KIND_LABEL: Record<FuelEntryKind, string> = { delivery: 'Delivery', reading: 'Reading' }

export function FuelPanel({ open, onClose, siteId, siteName }: FuelPanelProps) {
  const permissions = usePermissions()
  const canRecord = permissions.can('fuel.record')
  const canManage = permissions.can('fuel.manage')

  const statusQuery = useFuelStatus(open ? siteId : undefined)
  const recordMutation = useRecordFuelMutation()
  const voidMutation = useVoidFuelMutation()

  const [values, setValues] = useState<RecordFuelFormValues>(initialValues)
  const [touched, setTouched] = useState<Partial<Record<keyof RecordFuelFormValues, boolean>>>({})
  const [submitted, setSubmitted] = useState(false)
  const [clientKey, setClientKey] = useState(newClientKey)

  const [voidTarget, setVoidTarget] = useState<FuelEntry | null>(null)
  const [voidReason, setVoidReason] = useState('')
  const [voidTouched, setVoidTouched] = useState(false)

  // Reset the form every time the panel opens, not just on mount -- the
  // same panel instance stays alive across opens for different sites.
  // Adjusted synchronously during render rather than in an effect, per this
  // project's React Compiler lint rules (see SidePanel.tsx / CreateTicketModal.tsx).
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setValues(initialValues())
      setTouched({})
      setSubmitted(false)
      setClientKey(newClientKey())
    }
  }

  const fieldErrors = getFieldErrors(recordFuelSchema, values)

  function showError(field: keyof RecordFuelFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange<K extends keyof RecordFuelFormValues>(
    field: K,
    value: RecordFuelFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof RecordFuelFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    if (Object.keys(fieldErrors).length > 0) return

    try {
      const entry = await recordMutation.mutateAsync({
        siteId,
        kind: values.kind,
        litres: Number(values.litres),
        occurredAt: new Date(values.occurredAt).toISOString(),
        note: values.note || undefined,
        clientKey,
      })
      toast.success(values.kind === 'delivery' ? 'Delivery recorded.' : 'Reading recorded.')
      if (entry.variance) {
        const v = entry.variance
        const sign = v.differenceLitres >= 0 ? '+' : ''
        const message = `Estimate expected ${Math.round(v.estimatedBeforeLitres).toLocaleString()} L; this reading is ${sign}${Math.round(v.differenceLitres).toLocaleString()} L from that.`
        if (v.reviewWarning) toast.warning(message)
        else toast.success(message)
      }
      setValues(initialValues())
      setTouched({})
      setSubmitted(false)
      setClientKey(newClientKey())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Something went wrong. Try again.')
    }
  }

  function requestVoid(entry: FuelEntry) {
    setVoidTarget(entry)
    setVoidReason('')
    setVoidTouched(false)
  }

  async function confirmVoid() {
    if (!voidTarget) return
    if (voidReasonError(voidReason)) {
      setVoidTouched(true)
      return
    }
    try {
      await voidMutation.mutateAsync({ id: voidTarget.id, reason: voidReason })
      toast.success('Entry voided.')
      setVoidTarget(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Something went wrong. Try again.')
    }
  }

  const estimate = statusQuery.data?.estimate
  const entries = statusQuery.data?.entries ?? []

  const calculationRows: CalculationDetailRow[] = estimate
    ? [
        { label: 'Litres per running hour', value: `${estimate.consumptionLPH} L/h` },
        { label: 'Hours run since anchor', value: `${estimate.hoursRun}` },
        {
          label: 'Litres consumed since anchor',
          value: `${Math.round(estimate.consumedLitres).toLocaleString()} L`,
        },
        { label: 'Unreported days', value: `${estimate.unreportedDays}` },
        { label: 'As of', value: formatDateTimeWAT(estimate.asOf) },
      ]
    : []

  const calculationWarnings: string[] = estimate
    ? [
        estimate.overCapacity
          ? 'The estimate exceeds the tank capacity -- an entry is probably wrong.'
          : '',
        estimate.provisional
          ? 'Provisional: some days in this period have no approved generator hours.'
          : '',
        estimate.assumedEmptyStart
          ? 'Started from an assumed-empty tank -- no reading was available to anchor from.'
          : '',
      ].filter(Boolean)
    : []

  const columns: Column<FuelEntry>[] = [
    {
      header: 'Date',
      accessor: (entry) => entry.occurredAt,
      cell: (entry) => formatDateTimeWAT(entry.occurredAt),
      sortable: true,
    },
    {
      header: 'Kind',
      accessor: (entry) => KIND_LABEL[entry.kind],
      cell: (entry) => (
        <span className="inline-flex items-center gap-1.5">
          {KIND_LABEL[entry.kind]}
          {entry.voided && <StatusBadge label="Voided" tone="neutral" />}
        </span>
      ),
    },
    {
      header: 'Litres',
      accessor: (entry) => entry.litres,
      align: 'right',
      cell: (entry) => `${Math.round(entry.litres).toLocaleString()} L`,
    },
    {
      header: 'Variance',
      accessor: () => null,
      align: 'right',
      cell: (entry) => {
        if (!entry.variance) return <span className="text-text-muted">—</span>
        const sign = entry.variance.differenceLitres >= 0 ? '+' : ''
        return (
          <span
            className={
              entry.variance.reviewWarning ? 'font-medium text-warning-700' : 'text-text-muted'
            }
          >
            {sign}
            {Math.round(entry.variance.differenceLitres).toLocaleString()} L
          </span>
        )
      },
    },
    // recordedBy is only ever sent to staff (backend's IsStaff check) --
    // and this panel only renders in the ops portal, so showing it
    // unconditionally here is safe; the backend remains the actual gate.
    {
      header: 'Recorded by',
      accessor: (entry) => entry.recordedBy ?? '',
      cell: (entry) => (entry.recordedBy ? nameOrId(entry.recordedByName, entry.recordedBy) : '—'),
    },
    ...(canManage
      ? [
          {
            header: 'Actions',
            accessor: () => null,
            align: 'right' as const,
            cell: (entry: FuelEntry) => {
              const actions: RowAction[] = [
                {
                  label: 'Void',
                  icon: TrashIcon,
                  tone: 'danger',
                  disabled: entry.voided,
                  hint: entry.voided ? 'Already voided.' : undefined,
                  onClick: () => requestVoid(entry),
                },
              ]
              return <RowActions actions={actions} label="Entry actions" />
            },
          } satisfies Column<FuelEntry>,
        ]
      : []),
  ]

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow="Fuel"
      title={siteName}
      subtitle="Tank level, deliveries and readings for this site."
      maxWidth="2xl"
    >
      <div className="flex flex-col gap-6">
        <div>
          {statusQuery.isLoading ? (
            <SkeletonListItems count={1} withBadge={false} />
          ) : estimate ? (
            <div className="rounded-lg border border-border bg-surface p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-2xl font-bold text-text">
                    {estimate.available && estimate.litres != null
                      ? `${Math.round(estimate.litres).toLocaleString()} L`
                      : '—'}
                    {estimate.available && estimate.hoursRemaining != null && (
                      <span className="ml-2 text-base font-medium text-text-muted">
                        {Math.round(estimate.hoursRemaining)}h remaining
                      </span>
                    )}
                  </p>
                  {!estimate.available && (
                    <p className="mt-1 text-sm text-text-muted">
                      {estimate.reason || 'No estimate yet for this site.'}
                    </p>
                  )}
                </div>
                <BasisBadge basis="estimated" />
              </div>

              {estimate.available &&
                (estimate.low || estimate.estimatedEmpty || estimate.overCapacity) && (
                  <AlertBanner
                    className="mt-3"
                    tone={estimate.estimatedEmpty ? 'danger' : 'warning'}
                    icon={ExclamationTriangleIcon}
                    title={
                      estimate.estimatedEmpty
                        ? 'Estimated empty'
                        : estimate.low
                          ? 'Low fuel'
                          : 'Check this estimate'
                    }
                    description={
                      estimate.overCapacity
                        ? 'The estimate exceeds the tank capacity.'
                        : estimate.provisional
                          ? 'Provisional -- some days have no approved generator hours.'
                          : undefined
                    }
                  />
                )}

              {estimate.available && (
                <CalculationDetails
                  className="mt-3"
                  formula={estimate.formula}
                  formulaVersion={estimate.formulaVersion}
                  explanation={estimate.explanation}
                  rows={calculationRows}
                  warnings={calculationWarnings}
                />
              )}
            </div>
          ) : null}
        </div>

        {canRecord && (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4 border-t border-border pt-5">
            <h3 className="text-sm font-semibold text-text">Record a delivery or reading</h3>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Kind" htmlFor="fuel-kind">
                <Select
                  id="fuel-kind"
                  value={values.kind}
                  onChange={(event) => handleChange('kind', event.target.value as FuelEntryKind)}
                >
                  <option value="delivery">Delivery (litres added)</option>
                  <option value="reading">Reading (tank level now)</option>
                </Select>
              </Field>
              <Field label="Litres" htmlFor="fuel-litres" error={showError('litres')}>
                <Input
                  id="fuel-litres"
                  type="number"
                  min={0}
                  step={0.1}
                  inputMode="decimal"
                  invalid={Boolean(showError('litres'))}
                  value={values.litres}
                  onChange={(event) => handleChange('litres', event.target.value)}
                  onBlur={() => handleBlur('litres')}
                />
              </Field>
            </div>

            <Field label="When" htmlFor="fuel-occurred-at" error={showError('occurredAt')}>
              <Input
                id="fuel-occurred-at"
                type="datetime-local"
                max={nowLocalDatetimeValue()}
                invalid={Boolean(showError('occurredAt'))}
                value={values.occurredAt}
                onChange={(event) => handleChange('occurredAt', event.target.value)}
                onBlur={() => handleBlur('occurredAt')}
              />
            </Field>

            <Field
              label="Internal note"
              htmlFor="fuel-note"
              hint="(optional) Coloplus only -- never shown to the customer."
            >
              <Textarea
                id="fuel-note"
                value={values.note}
                onChange={(event) => handleChange('note', event.target.value)}
              />
            </Field>

            <Button type="submit" loading={recordMutation.isPending} className="self-start">
              {values.kind === 'delivery' ? 'Record delivery' : 'Record reading'}
            </Button>
          </form>
        )}

        <div className="border-t border-border pt-5">
          <h3 className="mb-3 text-sm font-semibold text-text">Entry history</h3>
          <DataTable
            data={entries}
            columns={columns}
            getRowId={(entry) => entry.id}
            loading={statusQuery.isLoading}
            compact
            paginated={false}
            searchable={false}
            emptyMessage="No deliveries or readings recorded yet."
            rowClassName={(entry) => (entry.voided ? 'opacity-60' : '')}
          />
        </div>
      </div>

      <ConfirmDialog
        open={Boolean(voidTarget)}
        title="Void this entry?"
        description="It stays in the record with your reason and no longer counts toward the estimate."
        tone="danger"
        confirmLabel="Void entry"
        pending={voidMutation.isPending}
        body={
          <Field
            label="Reason"
            htmlFor="fuel-void-reason"
            error={voidTouched ? voidReasonError(voidReason) : undefined}
          >
            <Textarea
              id="fuel-void-reason"
              value={voidReason}
              onChange={(event) => setVoidReason(event.target.value)}
              onBlur={() => setVoidTouched(true)}
              invalid={voidTouched && Boolean(voidReasonError(voidReason))}
            />
          </Field>
        }
        onConfirm={confirmVoid}
        onCancel={() => setVoidTarget(null)}
      />
    </SidePanel>
  )
}
