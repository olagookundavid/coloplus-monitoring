import { useState, type FormEvent } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { useCreateSiteMutation, useUpdateSiteMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import {
  POWER_SOURCES,
  POWER_SOURCE_LABELS,
  SITE_STATUSES,
  SITE_STATUS_LABELS,
  type PowerSource,
  type SiteProfile,
  type SiteProfileInput,
  type SiteStatus,
  type UnreportedTimePolicy,
} from '../../types'
import { getFieldErrors, siteProfileSchema, toast, type SiteProfileFormValues } from '../../utils'
import { Button, Checkbox, ConfirmDialog, ConfirmSummary, Field, Input, Select, SidePanel } from '../ui'

export interface SiteFormPanelProps {
  open: boolean
  onClose: () => void
  /** Present for edit; absent for a fresh site. */
  site?: SiteProfile
}

function minutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
    .toString()
    .padStart(2, '0')
  const m = (minutes % 60).toString().padStart(2, '0')
  return `${h}:${m}`
}

function timeToMinutes(value: string): number {
  const [h, m] = value.split(':').map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

function blankValues(): SiteProfileFormValues {
  return {
    airtelCode: '',
    coloplusCode: '',
    name: '',
    address: '',
    region: '',
    state: '',
    siteType: '',
    powerSources: [],
    responsibleTeam: '',
    unreportedTime: 'available',
    consumptionLPH: '',
    tankCapacityLitres: '',
    allDay: true,
    serviceStartTime: '',
    serviceEndTime: '',
    freshnessHours: '30',
    status: 'pending_acceptance',
    commissioningDate: '',
    latitude: '',
    longitude: '',
    generatorCapacity: '',
    accessContactName: '',
    accessContactPhone: '',
    supervisorName: '',
  }
}

function valuesFromSite(site: SiteProfile): SiteProfileFormValues {
  const allDay = site.serviceStartMinute === 0 && site.serviceEndMinute === 1440
  return {
    airtelCode: site.airtelCode,
    coloplusCode: site.coloplusCode,
    name: site.name,
    address: site.address,
    region: site.region,
    state: site.state,
    siteType: site.siteType,
    powerSources: site.powerSources,
    responsibleTeam: site.responsibleTeam,
    unreportedTime: site.unreportedTime,
    consumptionLPH: site.consumptionLPH != null ? String(site.consumptionLPH) : '',
    tankCapacityLitres: site.tankCapacityLitres != null ? String(site.tankCapacityLitres) : '',
    allDay,
    serviceStartTime: allDay ? '' : minutesToTime(site.serviceStartMinute),
    serviceEndTime: allDay ? '' : minutesToTime(site.serviceEndMinute),
    freshnessHours: String(site.freshnessHours),
    status: site.status,
    commissioningDate: site.commissioningDate ?? '',
    latitude: site.latitude != null ? String(site.latitude) : '',
    longitude: site.longitude != null ? String(site.longitude) : '',
    generatorCapacity: site.generatorCapacity,
    accessContactName: site.accessContactName,
    accessContactPhone: site.accessContactPhone,
    supervisorName: site.supervisorName,
  }
}

function toInput(values: SiteProfileFormValues): SiteProfileInput {
  return {
    airtelCode: values.airtelCode.trim(),
    coloplusCode: values.coloplusCode.trim(),
    name: values.name.trim(),
    address: values.address.trim(),
    region: values.region.trim(),
    state: values.state.trim(),
    siteType: values.siteType.trim(),
    powerSources: values.powerSources,
    responsibleTeam: values.responsibleTeam.trim(),
    unreportedTime: values.unreportedTime,
    consumptionLPH: values.consumptionLPH === '' ? undefined : Number(values.consumptionLPH),
    tankCapacityLitres:
      values.tankCapacityLitres === '' ? undefined : Number(values.tankCapacityLitres),
    serviceStartMinute: values.allDay ? 0 : timeToMinutes(values.serviceStartTime),
    serviceEndMinute: values.allDay ? 1440 : timeToMinutes(values.serviceEndTime),
    freshnessHours: Number(values.freshnessHours),
    status: values.status,
    commissioningDate: values.commissioningDate === '' ? undefined : values.commissioningDate,
    latitude: values.latitude === '' ? undefined : Number(values.latitude),
    longitude: values.longitude === '' ? undefined : Number(values.longitude),
    generatorCapacity: values.generatorCapacity.trim(),
    accessContactName: values.accessContactName.trim(),
    accessContactPhone: values.accessContactPhone.trim(),
    supervisorName: values.supervisorName.trim(),
  }
}

export function SiteFormPanel({ open, onClose, site }: SiteFormPanelProps) {
  const isEdit = Boolean(site)
  const createMutation = useCreateSiteMutation()
  const updateMutation = useUpdateSiteMutation()
  const pending = createMutation.isPending || updateMutation.isPending

  const [values, setValues] = useState<SiteProfileFormValues>(() =>
    site ? valuesFromSite(site) : blankValues()
  )
  const [touched, setTouched] = useState<Partial<Record<keyof SiteProfileFormValues, boolean>>>(
    {}
  )
  const [submitted, setSubmitted] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [pendingInput, setPendingInput] = useState<SiteProfileInput | null>(null)

  // Re-derive the form every time the panel opens, not just on mount -- the
  // same instance is reused across different sites/opens (same pattern as
  // CreateTicketModal.tsx / FuelPanel.tsx).
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setValues(site ? valuesFromSite(site) : blankValues())
      setTouched({})
      setSubmitted(false)
      setFormError(null)
    }
  }

  const fieldErrors = getFieldErrors(siteProfileSchema, values)

  function showError(field: keyof SiteProfileFormValues): string | undefined {
    return touched[field] || submitted ? fieldErrors[field] : undefined
  }

  function handleChange<K extends keyof SiteProfileFormValues>(
    field: K,
    value: SiteProfileFormValues[K]
  ) {
    setValues((current) => ({ ...current, [field]: value }))
  }

  function handleBlur(field: keyof SiteProfileFormValues) {
    setTouched((current) => ({ ...current, [field]: true }))
  }

  function togglePowerSource(source: PowerSource, checked: boolean) {
    setValues((current) => ({
      ...current,
      powerSources: checked
        ? [...current.powerSources, source]
        : current.powerSources.filter((s) => s !== source),
    }))
  }

  function handleClose() {
    onClose()
  }

  async function submit(input: SiteProfileInput) {
    try {
      if (isEdit && site) {
        await updateMutation.mutateAsync({ id: site.id, input })
      } else {
        await createMutation.mutateAsync(input)
      }
      toast.success(isEdit ? 'Site saved.' : `${input.name} created.`)
      setPendingInput(null)
      handleClose()
    } catch (error) {
      setPendingInput(null)
      setFormError(
        error instanceof AuthApiError ? error.message : 'Something went wrong. Try again.'
      )
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setFormError(null)
    if (Object.keys(fieldErrors).length > 0) return

    const input = toInput(values)
    const deactivating = isEdit && site?.status !== 'suspended' && input.status === 'suspended'
    if (deactivating) {
      setPendingInput(input)
      return
    }
    await submit(input)
  }

  return (
    <SidePanel
      open={open}
      onClose={handleClose}
      eyebrow={isEdit ? site?.airtelCode : 'New site'}
      title={isEdit ? site?.name || '' : 'Add a site'}
      subtitle={isEdit ? undefined : 'Codes must be unique across every site.'}
      maxWidth="2xl"
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="site-form" loading={pending}>
            {isEdit ? 'Save site' : 'Create site'}
          </Button>
        </>
      }
    >
      <form id="site-form" noValidate onSubmit={handleSubmit} className="flex flex-col gap-6">
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Identity</h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Airtel code" htmlFor="site-airtel-code" error={showError('airtelCode')}>
              <Input
                id="site-airtel-code"
                value={values.airtelCode}
                invalid={Boolean(showError('airtelCode'))}
                onChange={(event) => handleChange('airtelCode', event.target.value)}
                onBlur={() => handleBlur('airtelCode')}
              />
            </Field>
            <Field
              label="Coloplus code"
              htmlFor="site-coloplus-code"
              error={showError('coloplusCode')}
            >
              <Input
                id="site-coloplus-code"
                value={values.coloplusCode}
                invalid={Boolean(showError('coloplusCode'))}
                onChange={(event) => handleChange('coloplusCode', event.target.value)}
                onBlur={() => handleBlur('coloplusCode')}
              />
            </Field>
          </div>
          <Field label="Name" htmlFor="site-name" error={showError('name')}>
            <Input
              id="site-name"
              value={values.name}
              invalid={Boolean(showError('name'))}
              onChange={(event) => handleChange('name', event.target.value)}
              onBlur={() => handleBlur('name')}
            />
          </Field>
          <Field label="Type" htmlFor="site-type" hint="(optional) e.g. tower, rooftop, shelter">
            <Input
              id="site-type"
              value={values.siteType}
              onChange={(event) => handleChange('siteType', event.target.value)}
            />
          </Field>
        </section>

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Location</h3>
          <Field label="Address" htmlFor="site-address" hint="(optional)">
            <Input
              id="site-address"
              value={values.address}
              onChange={(event) => handleChange('address', event.target.value)}
            />
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Region" htmlFor="site-region" hint="(optional)">
              <Input
                id="site-region"
                value={values.region}
                onChange={(event) => handleChange('region', event.target.value)}
              />
            </Field>
            <Field label="State" htmlFor="site-state" error={showError('state')}>
              <Input
                id="site-state"
                value={values.state}
                invalid={Boolean(showError('state'))}
                onChange={(event) => handleChange('state', event.target.value)}
                onBlur={() => handleBlur('state')}
              />
            </Field>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field
              label="Latitude"
              htmlFor="site-latitude"
              hint="(optional)"
              error={showError('latitude')}
            >
              <Input
                id="site-latitude"
                type="number"
                step="any"
                invalid={Boolean(showError('latitude'))}
                value={values.latitude}
                onChange={(event) => handleChange('latitude', event.target.value)}
                onBlur={() => handleBlur('latitude')}
              />
            </Field>
            <Field
              label="Longitude"
              htmlFor="site-longitude"
              hint="(optional)"
              error={showError('longitude')}
            >
              <Input
                id="site-longitude"
                type="number"
                step="any"
                invalid={Boolean(showError('longitude'))}
                value={values.longitude}
                onChange={(event) => handleChange('longitude', event.target.value)}
                onBlur={() => handleBlur('longitude')}
              />
            </Field>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Status &amp; commissioning</h3>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Status" htmlFor="site-status">
              <Select
                id="site-status"
                value={values.status}
                onChange={(event) => handleChange('status', event.target.value as SiteStatus)}
              >
                {SITE_STATUSES.map((status) => (
                  <option key={status} value={status}>
                    {SITE_STATUS_LABELS[status]}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Commissioning date" htmlFor="site-commissioning-date" hint="(optional)">
              <Input
                id="site-commissioning-date"
                type="date"
                value={values.commissioningDate}
                onChange={(event) => handleChange('commissioningDate', event.target.value)}
              />
            </Field>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Power &amp; fuel</h3>
          <Field label="Power sources" error={showError('powerSources')}>
            <div className="flex flex-col gap-2">
              {POWER_SOURCES.map((source) => (
                <Checkbox
                  key={source}
                  label={POWER_SOURCE_LABELS[source]}
                  checked={values.powerSources.includes(source)}
                  onChange={(event) => togglePowerSource(source, event.target.checked)}
                />
              ))}
            </div>
          </Field>
          <Field label="Generator capacity" htmlFor="site-generator-capacity" hint="(optional) e.g. 20 kVA diesel">
            <Input
              id="site-generator-capacity"
              value={values.generatorCapacity}
              onChange={(event) => handleChange('generatorCapacity', event.target.value)}
            />
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field
              label="Litres per running hour"
              htmlFor="site-consumption-lph"
              hint="(optional)"
              error={showError('consumptionLPH')}
            >
              <Input
                id="site-consumption-lph"
                type="number"
                step="any"
                invalid={Boolean(showError('consumptionLPH'))}
                value={values.consumptionLPH}
                onChange={(event) => handleChange('consumptionLPH', event.target.value)}
                onBlur={() => handleBlur('consumptionLPH')}
              />
            </Field>
            <Field
              label="Tank capacity (litres)"
              htmlFor="site-tank-capacity"
              hint="(optional)"
              error={showError('tankCapacityLitres')}
            >
              <Input
                id="site-tank-capacity"
                type="number"
                step="any"
                invalid={Boolean(showError('tankCapacityLitres'))}
                value={values.tankCapacityLitres}
                onChange={(event) => handleChange('tankCapacityLitres', event.target.value)}
                onBlur={() => handleBlur('tankCapacityLitres')}
              />
            </Field>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Reporting rules</h3>
          <Checkbox
            label="Open all day"
            hint="Reports are accepted any time. Uncheck to set a specific service window."
            checked={values.allDay}
            onChange={(event) => handleChange('allDay', event.target.checked)}
          />
          {!values.allDay && (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field
                label="Service start"
                htmlFor="site-service-start"
                error={showError('serviceStartTime')}
              >
                <Input
                  id="site-service-start"
                  type="time"
                  invalid={Boolean(showError('serviceStartTime'))}
                  value={values.serviceStartTime}
                  onChange={(event) => handleChange('serviceStartTime', event.target.value)}
                  onBlur={() => handleBlur('serviceStartTime')}
                />
              </Field>
              <Field
                label="Service end"
                htmlFor="site-service-end"
                error={showError('serviceEndTime')}
              >
                <Input
                  id="site-service-end"
                  type="time"
                  invalid={Boolean(showError('serviceEndTime'))}
                  value={values.serviceEndTime}
                  onChange={(event) => handleChange('serviceEndTime', event.target.value)}
                  onBlur={() => handleBlur('serviceEndTime')}
                />
              </Field>
            </div>
          )}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field
              label="Freshness (hours)"
              htmlFor="site-freshness-hours"
              hint="How long an approved report keeps the site current."
              error={showError('freshnessHours')}
            >
              <Input
                id="site-freshness-hours"
                type="number"
                min={1}
                invalid={Boolean(showError('freshnessHours'))}
                value={values.freshnessHours}
                onChange={(event) => handleChange('freshnessHours', event.target.value)}
                onBlur={() => handleBlur('freshnessHours')}
              />
            </Field>
            <Field label="Unreported time reads as" htmlFor="site-unreported-time">
              <Select
                id="site-unreported-time"
                value={values.unreportedTime}
                onChange={(event) =>
                  handleChange('unreportedTime', event.target.value as UnreportedTimePolicy)
                }
              >
                <option value="available">Available (default)</option>
                <option value="unknown">Unknown</option>
              </Select>
            </Field>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <h3 className="text-sm font-semibold text-text">Contacts &amp; team</h3>
          <Field label="Responsible team" htmlFor="site-responsible-team" hint="(optional) shown to Airtel -- a team name, never a person">
            <Input
              id="site-responsible-team"
              value={values.responsibleTeam}
              onChange={(event) => handleChange('responsibleTeam', event.target.value)}
            />
          </Field>
          <Field label="Operations supervisor" htmlFor="site-supervisor-name" hint="(optional)">
            <Input
              id="site-supervisor-name"
              value={values.supervisorName}
              onChange={(event) => handleChange('supervisorName', event.target.value)}
            />
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Access contact name" htmlFor="site-access-contact-name" hint="(optional)">
              <Input
                id="site-access-contact-name"
                value={values.accessContactName}
                onChange={(event) => handleChange('accessContactName', event.target.value)}
              />
            </Field>
            <Field
              label="Access contact phone"
              htmlFor="site-access-contact-phone"
              hint="(optional)"
            >
              <Input
                id="site-access-contact-phone"
                value={values.accessContactPhone}
                onChange={(event) => handleChange('accessContactPhone', event.target.value)}
              />
            </Field>
          </div>
        </section>
      </form>

      <ConfirmDialog
        open={Boolean(pendingInput)}
        title="Deactivate this site?"
        description="Setting it to suspended stops it counting toward active-site figures. It can be reactivated here later."
        tone="warning"
        icon={ExclamationTriangleIcon}
        confirmLabel="Save as suspended"
        pending={pending}
        body={site && <ConfirmSummary title={site.name} meta={site.airtelCode} />}
        onConfirm={() => pendingInput && void submit(pendingInput)}
        onCancel={() => setPendingInput(null)}
      />
    </SidePanel>
  )
}
