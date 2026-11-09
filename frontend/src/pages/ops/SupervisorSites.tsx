import { useState } from 'react'
import { ClipboardDocumentCheckIcon, MapPinIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader, FuelPanel } from '../../components/dashboard'
import { SkeletonListItems } from '../../components/skeletons'
import {
  Button,
  ConfirmDialog,
  ConfirmSummary,
  DataTable,
  SidePanel,
  StatusBadge,
  Textarea,
  type Column,
  type StatusTone,
} from '../../components/ui'
import {
  useAdminSites,
  usePermissions,
  useRequestAcceptanceMutation,
  useSiteAcceptance,
  useSiteOverview,
  useTransitionAcceptanceMutation,
} from '../../hooks'
import { AuthApiError } from '../../services'
import type { AcceptanceStage, AdminSite } from '../../types'
import { acceptanceStageLabel, formatDateTimeWAT, severityLabel, toast } from '../../utils'

const NO_SITES: AdminSite[] = []

const CONDITION: Record<string, { label: string; tone: StatusTone }> = {
  available: { label: 'Available', tone: 'success' },
  outage_reported: { label: 'Outage reported', tone: 'danger' },
  awaiting_update: { label: 'Awaiting update', tone: 'warning' },
}

// The Coloplus side of the acceptance stage machine -- requested/built/
// ready_for_inspection/rejected-rework/hold/resume. The customer's side
// (inspected/accepted/rejected) is already built in pages/customer/
// Acceptance.tsx and SiteDetailPanel.tsx; this is the missing other half.
type PendingAcceptanceAction =
  | { kind: 'request' }
  | { kind: 'markBuilt'; note?: string }
  | { kind: 'submitForInspection'; note: string }
  | { kind: 'hold'; note: string }
  | { kind: 'resume'; toStage: AcceptanceStage }

const ACCEPTANCE_CONFIRM_COPY: Record<
  PendingAcceptanceAction['kind'],
  { title: string; description: string }
> = {
  request: {
    title: 'Start acceptance?',
    description: 'Begins the commissioning workflow for this site.',
  },
  markBuilt: {
    title: 'Mark this site built?',
    description: 'Moves acceptance to the next stage.',
  },
  submitForInspection: {
    title: 'Submit for inspection?',
    description: 'Sends the dossier to the customer for review.',
  },
  hold: {
    title: 'Put acceptance on hold?',
    description: 'Pauses the workflow until resumed.',
  },
  resume: {
    title: 'Resume acceptance?',
    description: 'Picks the workflow back up where it left off.',
  },
}

function SiteDetailBody({ site, onFuel }: { site: AdminSite; onFuel?: () => void }) {
  const overviewQuery = useSiteOverview(site.id)
  const overview = overviewQuery.data
  const condition = overview ? CONDITION[overview.condition] : undefined

  const { can } = usePermissions()
  const canManageAcceptance = can('acceptance.manage')

  const acceptanceQuery = useSiteAcceptance(site.id)
  const acceptance = acceptanceQuery.data?.acceptance
  const acceptanceEvents = acceptanceQuery.data?.events ?? []
  const noAcceptanceYet =
    acceptanceQuery.isError &&
    acceptanceQuery.error instanceof AuthApiError &&
    acceptanceQuery.error.status === 404

  const requestAcceptanceMutation = useRequestAcceptanceMutation()
  const transitionAcceptanceMutation = useTransitionAcceptanceMutation()
  const acceptanceMutationPending =
    requestAcceptanceMutation.isPending || transitionAcceptanceMutation.isPending

  const [pendingAcceptanceAction, setPendingAcceptanceAction] =
    useState<PendingAcceptanceAction | null>(null)
  const [reworkNote, setReworkNote] = useState('')
  const [dossierNote, setDossierNote] = useState('')
  const [holdNote, setHoldNote] = useState('')

  async function handleConfirmAcceptance() {
    if (!pendingAcceptanceAction) return
    try {
      switch (pendingAcceptanceAction.kind) {
        case 'request':
          await requestAcceptanceMutation.mutateAsync({ siteId: site.id })
          toast.success('Acceptance started.')
          break
        case 'markBuilt':
          await transitionAcceptanceMutation.mutateAsync({
            siteId: site.id,
            input: { stage: 'built', note: pendingAcceptanceAction.note },
          })
          toast.success('Marked built.')
          break
        case 'submitForInspection':
          await transitionAcceptanceMutation.mutateAsync({
            siteId: site.id,
            input: { stage: 'ready_for_inspection', note: pendingAcceptanceAction.note },
          })
          toast.success('Submitted for inspection.')
          break
        case 'hold':
          await transitionAcceptanceMutation.mutateAsync({
            siteId: site.id,
            input: { stage: 'on_hold', note: pendingAcceptanceAction.note },
          })
          toast.success('Put on hold.')
          break
        case 'resume':
          await transitionAcceptanceMutation.mutateAsync({
            siteId: site.id,
            input: { stage: pendingAcceptanceAction.toStage },
          })
          toast.success('Resumed.')
          break
      }
      setPendingAcceptanceAction(null)
      setReworkNote('')
      setDossierNote('')
      setHoldNote('')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update acceptance.')
      setPendingAcceptanceAction(null)
    }
  }

  return (
    <div className="flex flex-col gap-5">
      {onFuel && (
        <Button variant="outline" onClick={onFuel} className="self-start">
          Manage fuel
        </Button>
      )}

      <dl className="divide-y divide-border rounded-lg border border-border">
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Customer site code</dt>
          <dd className="font-medium text-text">{site.customerSiteCode}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Coloplus code</dt>
          <dd className="font-medium text-text">{site.coloplusCode}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Type</dt>
          <dd className="font-medium text-text">{site.siteType}</dd>
        </div>
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Region / State</dt>
          <dd className="font-medium text-text">
            {site.region} · {site.state}
          </dd>
        </div>
      </dl>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          Acceptance
        </p>
        {acceptanceQuery.isLoading ? (
          <SkeletonListItems count={1} withBadge={false} />
        ) : acceptance ? (
          <div className="flex flex-col gap-3 rounded-lg border border-border bg-surface p-4">
            <div className="flex items-center gap-2">
              {(() => {
                const { label, tone } = acceptanceStageLabel(acceptance.stage)
                return <StatusBadge label={label} tone={tone} />
              })()}
              {acceptance.overdue && <StatusBadge label="Overdue" tone="danger" />}
            </div>
            <p className="text-sm text-text-muted">
              Waiting on {acceptance.waitingOn === 'customer' ? 'Customer' : 'Coloplus'}
            </p>
            {acceptance.dueAt && (
              <p className={`text-sm ${acceptance.overdue ? 'text-danger' : 'text-text-muted'}`}>
                <span className="font-medium">Due: </span>
                {formatDateTimeWAT(acceptance.dueAt)}
                {acceptance.overdue ? ' (overdue)' : ''}
              </p>
            )}
            {acceptance.dossier && (
              <p className="text-sm text-text">
                <span className="font-medium">Dossier: </span>
                {acceptance.dossier}
              </p>
            )}
            {acceptance.reason && (
              <p className="text-sm text-danger-600">
                <span className="font-medium">Rejected: </span>
                {acceptance.reason}
              </p>
            )}
            {acceptance.holdReason && (
              <p className="text-sm text-warning-700">
                <span className="font-medium">On hold: </span>
                {acceptance.holdReason}
              </p>
            )}

            {canManageAcceptance && acceptance.stage === 'requested' && (
              <Button
                variant="secondary"
                onClick={() => setPendingAcceptanceAction({ kind: 'markBuilt' })}
              >
                Mark built
              </Button>
            )}

            {canManageAcceptance && acceptance.stage === 'rejected' && (
              <div className="flex flex-col gap-2">
                <Textarea
                  value={reworkNote}
                  onChange={(event) => setReworkNote(event.target.value)}
                  placeholder="What was fixed? (required)"
                  aria-label="What was fixed"
                />
                <Button
                  variant="secondary"
                  disabled={!reworkNote.trim()}
                  onClick={() =>
                    setPendingAcceptanceAction({ kind: 'markBuilt', note: reworkNote.trim() })
                  }
                >
                  Mark built
                </Button>
              </div>
            )}

            {canManageAcceptance && acceptance.stage === 'built' && (
              <div className="flex flex-col gap-2">
                <Textarea
                  value={dossierNote}
                  onChange={(event) => setDossierNote(event.target.value)}
                  placeholder="Dossier: what's built and tested, and where the evidence is (required)"
                  aria-label="Dossier"
                />
                <Button
                  disabled={!dossierNote.trim()}
                  onClick={() =>
                    setPendingAcceptanceAction({
                      kind: 'submitForInspection',
                      note: dossierNote.trim(),
                    })
                  }
                >
                  Submit for inspection
                </Button>
              </div>
            )}

            {canManageAcceptance && acceptance.stage === 'on_hold' && acceptance.heldFrom && (
              <Button
                variant="secondary"
                onClick={() =>
                  setPendingAcceptanceAction({
                    kind: 'resume',
                    toStage: acceptance.heldFrom as AcceptanceStage,
                  })
                }
              >
                Resume
              </Button>
            )}

            {canManageAcceptance &&
              acceptance.stage !== 'accepted' &&
              acceptance.stage !== 'on_hold' && (
                <div className="flex flex-col gap-2 border-t border-border pt-3">
                  <Textarea
                    value={holdNote}
                    onChange={(event) => setHoldNote(event.target.value)}
                    placeholder="Reason for the hold (required)"
                    aria-label="Hold reason"
                  />
                  <Button
                    variant="outline"
                    disabled={!holdNote.trim()}
                    onClick={() =>
                      setPendingAcceptanceAction({ kind: 'hold', note: holdNote.trim() })
                    }
                  >
                    Put on hold
                  </Button>
                </div>
              )}

            {canManageAcceptance &&
              (acceptance.stage === 'ready_for_inspection' || acceptance.stage === 'inspected') && (
                <p className="text-xs text-text-muted">Waiting on the customer to act next.</p>
              )}

            <div className="border-t border-border pt-3">
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
                Timeline
              </p>
              {acceptanceEvents.length === 0 ? (
                <p className="text-sm text-text-muted">No activity yet.</p>
              ) : (
                <ul className="flex flex-col gap-2 text-sm">
                  {acceptanceEvents.map((event) => (
                    <li key={event.id} className="flex gap-3">
                      <span className="w-32 shrink-0 text-text-muted">
                        {formatDateTimeWAT(event.createdAt)}
                      </span>
                      <span className="text-text">{event.body || `Moved to ${event.toStage}`}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        ) : noAcceptanceYet ? (
          canManageAcceptance ? (
            <Button
              variant="outline"
              onClick={() => setPendingAcceptanceAction({ kind: 'request' })}
            >
              Start acceptance
            </Button>
          ) : (
            <p className="text-sm text-text-muted">Acceptance hasn't started for this site.</p>
          )
        ) : (
          <p className="text-sm text-text-muted">Could not load the acceptance record.</p>
        )}
      </div>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          Live condition
        </p>
        {overviewQuery.isLoading ? (
          <SkeletonListItems count={1} withBadge={false} />
        ) : condition ? (
          <div className="flex items-center justify-between rounded-lg border border-border bg-surface p-4">
            <StatusBadge label={condition.label} tone={condition.tone} />
            {overview?.asOf && (
              <span className="text-xs text-text-muted">as of {formatDateTimeWAT(overview.asOf)}</span>
            )}
          </div>
        ) : (
          <p className="text-sm text-text-muted">No report yet.</p>
        )}
      </div>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
          Open incidents ({overview?.openIncidents ?? 0})
        </p>
        {overviewQuery.isLoading ? (
          <SkeletonListItems count={2} />
        ) : (overview?.incidents.length ?? 0) === 0 ? (
          <p className="text-sm text-text-muted">No open incidents at this site.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {overview?.incidents.map((incident) => {
              const { label, tone } = severityLabel(incident.severity)
              return (
                <li
                  key={incident.id}
                  className="rounded-lg border border-border bg-surface p-3 text-sm"
                >
                  <div className="flex items-center justify-between gap-2">
                    <StatusBadge label={label} tone={tone} />
                    <span className="text-xs text-text-muted">{incident.ref}</span>
                  </div>
                  <p className="mt-1.5 font-medium text-text">{incident.title}</p>
                </li>
              )
            })}
          </ul>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(pendingAcceptanceAction)}
        title={pendingAcceptanceAction ? ACCEPTANCE_CONFIRM_COPY[pendingAcceptanceAction.kind].title : ''}
        description={
          pendingAcceptanceAction
            ? ACCEPTANCE_CONFIRM_COPY[pendingAcceptanceAction.kind].description
            : ''
        }
        tone={pendingAcceptanceAction?.kind === 'hold' ? 'warning' : 'success'}
        icon={ClipboardDocumentCheckIcon}
        confirmLabel="Confirm"
        pending={acceptanceMutationPending}
        body={<ConfirmSummary title={site.name} meta={site.customerSiteCode} />}
        onConfirm={() => void handleConfirmAcceptance()}
        onCancel={() => setPendingAcceptanceAction(null)}
      />
    </div>
  )
}

export function SupervisorSites() {
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const permissions = usePermissions()
  const canSeeFuel = permissions.can(['fuel.record', 'fuel.read'])
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)
  const selectedSite = sites.find((site) => site.id === selectedSiteId) ?? null
  const [fuelSite, setFuelSite] = useState<AdminSite | null>(null)

  const columns: Column<AdminSite>[] = [
    { header: 'Site', accessor: (site) => site.customerSiteCode },
    {
      header: 'Name and type',
      accessor: (site) => site.name,
      cell: (site) => (
        <div>
          <p className="font-medium text-text">{site.name}</p>
          <p className="text-xs text-text-muted">{site.siteType}</p>
        </div>
      ),
    },
    { header: 'Region', accessor: (site) => site.region },
    { header: 'State', accessor: (site) => site.state },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Sites"
        subtitle="Every site in your scope. Open one for its live condition and open incidents."
      />

      <DataTable
        data={sites}
        columns={columns}
        getRowId={(site) => site.id}
        loading={sitesQuery.isLoading}
        onRowClick={(site) => setSelectedSiteId(site.id)}
        searchPlaceholder="Search site code or name..."
        emptyMessage="No sites in your scope yet."
      />

      <SidePanel
        open={Boolean(selectedSite)}
        onClose={() => setSelectedSiteId(null)}
        eyebrow={selectedSite?.customerSiteCode}
        title={selectedSite?.name ?? ''}
        subtitle={
          selectedSite && (
            <span className="inline-flex items-center gap-1.5">
              <MapPinIcon className="h-4 w-4" aria-hidden="true" />
              {selectedSite.region}, {selectedSite.state}
            </span>
          )
        }
      >
        {selectedSite && (
          <SiteDetailBody
            key={selectedSite.id}
            site={selectedSite}
            onFuel={canSeeFuel ? () => setFuelSite(selectedSite) : undefined}
          />
        )}
      </SidePanel>

      {fuelSite && (
        <FuelPanel
          open
          onClose={() => setFuelSite(null)}
          siteId={fuelSite.id}
          siteName={fuelSite.name}
        />
      )}
    </div>
  )
}
