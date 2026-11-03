import { useMemo, useState } from 'react'
import { CheckCircleIcon, PlusIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, LogIncidentModal } from '../../components/dashboard'
import { SkeletonDetailPane, SkeletonListItems } from '../../components/skeletons'
import {
  AttachmentList,
  Button,
  ConfirmDialog,
  ConfirmSummary,
  Field,
  FileDropZone,
  Input,
  Spinner,
  StatusBadge,
  Tabs,
  type TabItem,
} from '../../components/ui'
import {
  useAddIncidentUpdateMutation,
  useAttachments,
  useMySites,
  usePermissions,
  useReporterIncidentDetail,
  useReporterIncidents,
  useRestoreIncidentMutation,
} from '../../hooks'
import type { IncidentUpdateEntry, MySiteRow, ReporterIncident } from '../../types'
import { describeIncidentStatus, formatAuditWhen, severityLabel, toast } from '../../utils'
import type { IncidentBucket } from '../../utils'

const NO_INCIDENTS: ReporterIncident[] = []
const NO_SITES: MySiteRow[] = []

type TabValue = IncidentBucket | 'all'

function siteLabelFor(sitesById: Map<string, MySiteRow>, siteId: string): string {
  const site = sitesById.get(siteId)
  return site ? site.airtelCode : siteId
}

// Falls back to a plain description of the status change for an
// auto-generated timeline row that carries no note (e.g. a bare restore with
// no "note" field filled in).
function describeUpdate(update: IncidentUpdateEntry): string {
  if (update.body) return update.body
  if (update.kind === 'status' && update.toStatus) {
    return `Status changed to ${update.toStatus.replace(/_/g, ' ')}`
  }
  return 'Update recorded'
}

function combineTodayWithTime(time: string): string {
  const [h, m] = time.split(':').map(Number)
  const d = new Date()
  d.setHours(h ?? 0, m ?? 0, 0, 0)
  return d.toISOString()
}

interface IncidentDetailPaneProps {
  incident: ReporterIncident
  siteLabel: string
  onRequestRestore: (incident: ReporterIncident, atIso: string) => void
}

function IncidentDetailPane({ incident, siteLabel, onRequestRestore }: IncidentDetailPaneProps) {
  const detailQuery = useReporterIncidentDetail(incident.id)
  const updates = detailQuery.data?.updates ?? []
  const addUpdateMutation = useAddIncidentUpdateMutation()

  const { can } = usePermissions()
  const canAttach = can(['incidents.create', 'incidents.manage'])
  const isStaff = !can('published.read')
  const attachmentsQuery = useAttachments('incident', incident.id)

  const [updateText, setUpdateText] = useState('')
  const [restoredAtTime, setRestoredAtTime] = useState('')

  const {
    bucket,
    label: statusLabelText,
    tone: statusTone,
  } = describeIncidentStatus(incident.status)
  const { label: sevLabel, tone: sevTone } = severityLabel(incident.severity)

  async function handleAddUpdate() {
    const body = updateText.trim()
    if (!body) return
    try {
      await addUpdateMutation.mutateAsync({ id: incident.id, body })
      setUpdateText('')
      toast.success('Update added.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not add this update.')
    }
  }

  function handleSubmitForReview() {
    if (!restoredAtTime) return
    onRequestRestore(incident, combineTodayWithTime(restoredAtTime))
  }

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge label={sevLabel} tone={sevTone} />
        <span className="text-sm text-text-muted">{incident.ref}</span>
        <StatusBadge label={statusLabelText} tone={statusTone} />
      </div>

      <div>
        <h2 className="text-xl font-bold text-text">
          {incident.title} · {siteLabel}
        </h2>
        <p className="mt-1 text-sm text-text-muted">{incident.description}</p>
      </div>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Timeline</p>
        {detailQuery.isLoading ? (
          <Spinner className="mt-3 h-5 w-5" />
        ) : updates.length === 0 ? (
          <p className="mt-3 text-sm text-text-muted">No updates yet.</p>
        ) : (
          <ul className="mt-3 flex flex-col gap-3">
            {updates.map((update) => (
              <li key={update.id} className="flex gap-3 text-sm">
                <span className="w-14 shrink-0 text-text-muted">
                  {formatAuditWhen(update.observedAt ?? update.createdAt).time}
                </span>
                <span className="text-text">{describeUpdate(update)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex flex-col gap-3">
        <AttachmentList
          attachments={attachmentsQuery.data ?? []}
          loading={attachmentsQuery.isLoading}
          emptyMessage="No evidence attached yet."
        />
        {bucket === 'open' && canAttach && (
          <FileDropZone
            ownerType="incident"
            ownerId={incident.id}
            isStaff={isStaff}
            label="Add evidence"
          />
        )}
      </div>

      {bucket === 'open' && (
        <>
          <div>
            <p className="text-sm font-medium text-text">Add an update</p>
            <div className="mt-2 flex gap-2">
              <Input
                value={updateText}
                onChange={(event) => setUpdateText(event.target.value)}
                placeholder="e.g. Engineer on site, parts ordered"
                aria-label="Add an update"
              />
              <Button
                variant="secondary"
                onClick={handleAddUpdate}
                disabled={!updateText.trim() || addUpdateMutation.isPending}
                loading={addUpdateMutation.isPending}
              >
                Add
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-surface-subtle p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">
              Mark as restored
            </p>
            <div className="mt-3 flex flex-col gap-4">
              <Field label="Restored at" htmlFor="incident-restored-at">
                <Input
                  id="incident-restored-at"
                  type="time"
                  value={restoredAtTime}
                  onChange={(event) => setRestoredAtTime(event.target.value)}
                />
              </Field>
              <Button onClick={handleSubmitForReview} disabled={!restoredAtTime} fullWidth>
                Submit for review
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

export function ReporterIncidents() {
  const incidentsQuery = useReporterIncidents()
  const incidents = incidentsQuery.data ?? NO_INCIDENTS
  const sitesQuery = useMySites()
  const sites = sitesQuery.data?.sites ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.siteId, site])), [sites])

  const [tab, setTab] = useState<TabValue>('open')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [pendingRestore, setPendingRestore] = useState<{
    incident: ReporterIncident
    atIso: string
  } | null>(null)

  const restoreMutation = useRestoreIncidentMutation()

  const counts = useMemo(() => {
    const result = { open: 0, awaitingReview: 0, closed: 0 }
    for (const incident of incidents) {
      result[describeIncidentStatus(incident.status).bucket]++
    }
    return result
  }, [incidents])

  const visibleIncidents = useMemo(
    () =>
      tab === 'all'
        ? incidents
        : incidents.filter((incident) => describeIncidentStatus(incident.status).bucket === tab),
    [incidents, tab]
  )

  const selectedIncident =
    visibleIncidents.find((incident) => incident.id === selectedId) ?? visibleIncidents[0] ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'open', label: `Open ${counts.open}` },
    { value: 'awaitingReview', label: `Awaiting review ${counts.awaitingReview}` },
    { value: 'closed', label: `Closed ${counts.closed}` },
    { value: 'all', label: `All ${incidents.length}` },
  ]

  async function handleConfirmRestore() {
    if (!pendingRestore) return
    try {
      await restoreMutation.mutateAsync({
        id: pendingRestore.incident.id,
        at: pendingRestore.atIso,
      })
      toast.success('Incident marked restored.')
      setPendingRestore(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not mark this incident restored.')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Incidents"
        subtitle="Log outages, faults and security events at your sites. Close one by recording the restoration time and evidence."
        actions={
          <Button icon={PlusIcon} onClick={() => setCreateOpen(true)}>
            Log incident
          </Button>
        }
      />

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {incidentsQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : visibleIncidents.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              visibleIncidents.map((incident) => {
                const selected = incident.id === (selectedIncident?.id ?? null)
                const { label: sevLabel, tone: sevTone } = severityLabel(incident.severity)
                const { label: statusText, tone: statusTone } = describeIncidentStatus(
                  incident.status
                )
                const { day } = formatAuditWhen(incident.startedAt)
                return (
                  <li key={incident.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(incident.id)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        selected
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <StatusBadge label={sevLabel} tone={sevTone} />
                        <span className="shrink-0 text-xs text-text-muted">{incident.ref}</span>
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {incident.title} · {siteLabelFor(sitesById, incident.siteId)}
                      </p>
                      <p className="mt-1 text-sm text-text-muted">
                        {day} · <StatusBadge label={statusText} tone={statusTone} />
                      </p>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {incidentsQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedIncident ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select an incident to see its details.
            </div>
          ) : (
            <IncidentDetailPane
              key={selectedIncident.id}
              incident={selectedIncident}
              siteLabel={siteLabelFor(sitesById, selectedIncident.siteId)}
              onRequestRestore={(incident, atIso) => setPendingRestore({ incident, atIso })}
            />
          )}
        </div>
      </div>

      <LogIncidentModal open={createOpen} onClose={() => setCreateOpen(false)} />

      <ConfirmDialog
        open={Boolean(pendingRestore)}
        title="Mark this incident restored?"
        description="This moves it to awaiting review. A supervisor will verify and close it from here."
        tone="success"
        icon={CheckCircleIcon}
        confirmLabel="Submit for review"
        pending={restoreMutation.isPending}
        body={
          pendingRestore && (
            <ConfirmSummary
              title={`${pendingRestore.incident.title} · ${siteLabelFor(sitesById, pendingRestore.incident.siteId)}`}
              meta={pendingRestore.incident.ref}
            />
          )
        }
        onConfirm={handleConfirmRestore}
        onCancel={() => setPendingRestore(null)}
      />
    </div>
  )
}
