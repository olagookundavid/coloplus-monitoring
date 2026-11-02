import { useState, type FormEvent } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { Button, Field, Input, Select, SidePanel, Textarea } from '../ui'
import { useCreateIncidentMutation, useMySites } from '../../hooks'
import type { IncidentCategory, IncidentImpact, IncidentSeverity, MySiteRow } from '../../types'
import { severityLabel, toast } from '../../utils'

export interface LogIncidentModalProps {
  open: boolean
  onClose: () => void
}

const CATEGORY_LABELS: Record<IncidentCategory, string> = {
  grid_loss: 'Grid power loss',
  building_supply_loss: 'Building supply loss',
  generator_failure: 'Generator failure',
  low_fuel: 'Low fuel',
  battery_failure: 'Battery failure',
  rectifier_fault: 'Rectifier fault',
  electrical_fault: 'Electrical fault',
  access_restriction: 'Access issue',
  other: 'Other',
}

const CATEGORY_OPTIONS = Object.keys(CATEGORY_LABELS) as IncidentCategory[]
const SEVERITY_OPTIONS: IncidentSeverity[] = ['minor', 'major', 'critical']

const IMPACT_LABELS: Record<IncidentImpact, string> = {
  outage: 'Caused an outage',
  no_outage: 'No service interruption',
}
const IMPACT_OPTIONS: IncidentImpact[] = ['outage', 'no_outage']

function nowAsDatetimeLocal(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const NO_SITES: MySiteRow[] = []

export function LogIncidentModal({ open, onClose }: LogIncidentModalProps) {
  const sitesQuery = useMySites()
  const sites = sitesQuery.data?.sites ?? NO_SITES
  const mutation = useCreateIncidentMutation()

  const [siteId, setSiteId] = useState('')
  const [category, setCategory] = useState<IncidentCategory>('grid_loss')
  const [severity, setSeverity] = useState<IncidentSeverity>('major')
  const [impact, setImpact] = useState<IncidentImpact>('outage')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [startedAt, setStartedAt] = useState(nowAsDatetimeLocal())
  const [error, setError] = useState<string | null>(null)

  function resetForm() {
    setSiteId('')
    setCategory('grid_loss')
    setSeverity('major')
    setImpact('outage')
    setTitle('')
    setDescription('')
    setStartedAt(nowAsDatetimeLocal())
    setError(null)
  }

  function handleClose() {
    onClose()
    resetForm()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!siteId) {
      setError('Choose which site this happened at.')
      return
    }
    if (!title.trim()) {
      setError('Give the incident a short title.')
      return
    }
    if (!description.trim()) {
      setError('Describe what happened.')
      return
    }
    if (!startedAt) {
      setError('Say when the fault began.')
      return
    }
    const startedIso = new Date(startedAt).toISOString()
    if (new Date(startedIso).getTime() > Date.now()) {
      setError('The start time cannot be in the future.')
      return
    }
    setError(null)

    try {
      await mutation.mutateAsync({
        siteId,
        category,
        severity,
        impact,
        title: title.trim(),
        description: description.trim(),
        startedAt: startedIso,
      })
      toast.success('Incident logged.')
      handleClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not log this incident.')
    }
  }

  return (
    <SidePanel
      open={open}
      onClose={handleClose}
      eyebrow="Incidents"
      title="Log an incident"
      subtitle="Report a fault at one of your sites. You can add updates and record restoration afterwards."
      footer={
        <>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="log-incident-form" loading={mutation.isPending}>
            Log incident
          </Button>
        </>
      }
    >
      <form
        id="log-incident-form"
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

        <Field label="Site" htmlFor="incident-site">
          <Select id="incident-site" value={siteId} onChange={(event) => setSiteId(event.target.value)}>
            <option value="">Select a site</option>
            {sites.map((site) => (
              <option key={site.siteId} value={site.siteId}>
                {site.airtelCode} · {site.name}
              </option>
            ))}
          </Select>
        </Field>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Category" htmlFor="incident-category">
            <Select
              id="incident-category"
              value={category}
              onChange={(event) => setCategory(event.target.value as IncidentCategory)}
            >
              {CATEGORY_OPTIONS.map((c) => (
                <option key={c} value={c}>
                  {CATEGORY_LABELS[c]}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Severity" htmlFor="incident-severity">
            <Select
              id="incident-severity"
              value={severity}
              onChange={(event) => setSeverity(event.target.value as IncidentSeverity)}
            >
              {SEVERITY_OPTIONS.map((s) => (
                <option key={s} value={s}>
                  {severityLabel(s).label}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <Field label="Impact" htmlFor="incident-impact">
          <Select
            id="incident-impact"
            value={impact}
            onChange={(event) => setImpact(event.target.value as IncidentImpact)}
          >
            {IMPACT_OPTIONS.map((i) => (
              <option key={i} value={i}>
                {IMPACT_LABELS[i]}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Title" htmlFor="incident-title">
          <Input
            id="incident-title"
            placeholder="Briefly describe the incident"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
        </Field>

        <Field label="Description" htmlFor="incident-description">
          <Textarea
            id="incident-description"
            placeholder="What happened, and what you've seen so far"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>

        <Field label="Started at" htmlFor="incident-started" hint="When the fault began, as best you know.">
          <Input
            id="incident-started"
            type="datetime-local"
            value={startedAt}
            onChange={(event) => setStartedAt(event.target.value)}
          />
        </Field>
      </form>
    </SidePanel>
  )
}
