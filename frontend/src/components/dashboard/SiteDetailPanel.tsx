import { useState } from 'react'
import { MapPinIcon, PlusIcon } from '@heroicons/react/24/outline'

import { useSiteOverview, useTransitionAcceptanceMutation, usePermissions } from '../../hooks'
import type { PortfolioSite } from '../../types'
import {
  acceptanceStageLabel,
  disputeStatusLabel,
  formatDateTimeWAT,
  severityLabel,
  siteConditionLabel,
  toast,
} from '../../utils'
import { Button, ConfirmDialog, SidePanel, StatusBadge, Tabs, Textarea, type TabItem } from '../ui'
import { RaiseDisputeModal } from './RaiseDisputeModal'

export interface SiteDetailPanelProps {
  site: PortfolioSite | null
  open: boolean
  onClose: () => void
  onCreateTicket: () => void
}

type SiteDetailTab = 'profile' | 'incidents' | 'disputes' | 'acceptance'

const TABS: TabItem<SiteDetailTab>[] = [
  { value: 'profile', label: 'Profile' },
  { value: 'incidents', label: 'Incidents' },
  { value: 'disputes', label: 'Disputes' },
  { value: 'acceptance', label: 'Acceptance' },
]

type PendingAcceptanceAction = { stage: 'inspected' | 'accepted' | 'rejected'; reason?: string }

export function SiteDetailPanel({ site, open, onClose, onCreateTicket }: SiteDetailPanelProps) {
  const [tab, setTab] = useState<SiteDetailTab>('profile')
  const [disputeModalOpen, setDisputeModalOpen] = useState(false)
  const [rejectReason, setRejectReason] = useState('')
  const [pendingAction, setPendingAction] = useState<PendingAcceptanceAction | null>(null)

  const { can } = usePermissions()
  const canReview = can('disputes.raise')
  const canReviewAcceptance = can('acceptance.review')

  const overviewQuery = useSiteOverview(site?.id)
  const overview = overviewQuery.data
  const transitionMutation = useTransitionAcceptanceMutation()

  if (!site) return null

  const condition = siteConditionLabel(site.condition)
  const acceptance = overview?.acceptance
  const canActOnAcceptance =
    canReviewAcceptance && acceptance?.waitingOn === 'airtel' && !overviewQuery.isLoading

  async function handleConfirmAcceptance() {
    if (!pendingAction || !site) return
    try {
      await transitionMutation.mutateAsync({
        siteId: site.id,
        input: { stage: pendingAction.stage, note: pendingAction.reason },
      })
      toast.success(
        pendingAction.stage === 'accepted'
          ? 'Site accepted.'
          : pendingAction.stage === 'rejected'
            ? 'Site rejected.'
            : 'Marked as inspected.'
      )
      setPendingAction(null)
      setRejectReason('')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update this acceptance record.')
      setPendingAction(null)
    }
  }

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow={`${site.airtelCode} · ${site.region.toUpperCase()}`}
      title={site.name}
      headerActions={
        <Button icon={PlusIcon} size="sm" onClick={onCreateTicket}>
          Create ticket
        </Button>
      }
      subtitle={
        <div className="flex flex-col gap-2">
          <span className="inline-flex items-center gap-1.5">
            <MapPinIcon className="h-4 w-4 shrink-0 text-text-muted" aria-hidden="true" />
            {site.state}, Nigeria
          </span>
          <StatusBadge label={condition.label} tone={condition.tone} />
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <Tabs aria-label="Site detail" items={TABS} value={tab} onChange={setTab} />

        {tab === 'profile' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Site details</h3>
            <dl className="mt-4 divide-y divide-border">
              {overview?.site && (
                <>
                  <ProfileRow label="Address" value={overview.site.address || '—'} />
                  <ProfileRow label="Site type" value={overview.site.siteType} />
                  <ProfileRow
                    label="Power sources"
                    value={overview.site.powerSources.join(', ') || '—'}
                  />
                  <ProfileRow label="Generator capacity" value={overview.site.generatorCapacity || '—'} />
                  <ProfileRow
                    label="Commissioning date"
                    value={overview.site.commissioningDate ?? '—'}
                  />
                </>
              )}
              {site.asOf && (
                <ProfileRow label="Last power observation" value={formatDateTimeWAT(site.asOf)} />
              )}
            </dl>
          </div>
        )}

        {tab === 'incidents' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Fault & restoration history</h3>
            {overviewQuery.isLoading ? (
              <p className="mt-3 text-sm text-text-muted">Loading...</p>
            ) : (overview?.publishedIncidents.length ?? 0) === 0 ? (
              <p className="mt-3 text-sm text-text-muted">No incidents recorded for this site.</p>
            ) : (
              <div className="mt-4 flex flex-col gap-4">
                {overview?.publishedIncidents.map((incident) => {
                  const { label, tone } = severityLabel(incident.severity)
                  return (
                    <div key={incident.id} className="flex gap-3">
                      <span
                        className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary"
                        aria-hidden="true"
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <StatusBadge label={label} tone={tone} />
                          <span className="text-xs text-text-muted">{incident.ref}</span>
                        </div>
                        <p className="mt-1 font-semibold text-text">{incident.title}</p>
                        <p className="mt-1 text-xs text-text-muted">
                          Started {formatDateTimeWAT(incident.startedAt)}
                          {incident.closedAt && ` · Closed ${formatDateTimeWAT(incident.closedAt)}`}
                        </p>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}

        {tab === 'disputes' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-base font-semibold text-text">Disputes for this site</h3>
              {canReview && (
                <Button size="sm" variant="outline" onClick={() => setDisputeModalOpen(true)}>
                  Raise a dispute
                </Button>
              )}
            </div>
            {overviewQuery.isLoading ? (
              <p className="mt-3 text-sm text-text-muted">Loading...</p>
            ) : (overview?.disputes.length ?? 0) === 0 ? (
              <p className="mt-3 text-sm text-text-muted">No disputes raised for this site.</p>
            ) : (
              <div className="mt-4 flex flex-col divide-y divide-border">
                {overview?.disputes.map((dispute) => {
                  const { label, tone } = disputeStatusLabel(dispute.status)
                  return (
                    <div key={dispute.id} className="flex items-start justify-between gap-3 py-3">
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-primary-600">{dispute.ref}</p>
                        <p className="font-medium text-text">{dispute.title}</p>
                      </div>
                      <StatusBadge label={label} tone={tone} className="shrink-0" />
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}

        {tab === 'acceptance' && (
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h3 className="text-base font-semibold text-text">Commissioning and acceptance</h3>
            {overviewQuery.isLoading ? (
              <p className="mt-3 text-sm text-text-muted">Loading...</p>
            ) : !acceptance ? (
              <p className="mt-3 text-sm text-text-muted">No acceptance process for this site.</p>
            ) : (
              <>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <StatusBadge
                    label={acceptanceStageLabel(acceptance.stage).label}
                    tone={acceptanceStageLabel(acceptance.stage).tone}
                  />
                  {acceptance.overdue && <StatusBadge label="Overdue" tone="danger" />}
                </div>
                <dl className="mt-4 divide-y divide-border">
                  <ProfileRow
                    label="Waiting on"
                    value={acceptance.waitingOn === 'airtel' ? 'Airtel' : 'Coloplus'}
                  />
                  {acceptance.nextAction && (
                    <ProfileRow label="Next action" value={acceptance.nextAction} />
                  )}
                  <ProfileRow
                    label="Ready for inspection"
                    value={`${acceptance.readyAgeHours}h ago (not operational performance)`}
                  />
                  {acceptance.dossier && <ProfileRow label="Dossier" value={acceptance.dossier} />}
                </dl>

                {canActOnAcceptance && acceptance.stage === 'ready_for_inspection' && (
                  <Button
                    className="mt-4"
                    onClick={() => setPendingAction({ stage: 'inspected' })}
                  >
                    Mark inspected
                  </Button>
                )}

                {canActOnAcceptance && acceptance.stage === 'inspected' && (
                  <div className="mt-4 flex flex-col gap-3">
                    <Button onClick={() => setPendingAction({ stage: 'accepted' })} fullWidth>
                      Accept
                    </Button>
                    <Textarea
                      value={rejectReason}
                      onChange={(event) => setRejectReason(event.target.value)}
                      placeholder="Reason for rejecting (required)"
                      aria-label="Rejection reason"
                    />
                    <Button
                      variant="outline"
                      disabled={!rejectReason.trim()}
                      onClick={() =>
                        setPendingAction({ stage: 'rejected', reason: rejectReason.trim() })
                      }
                      fullWidth
                    >
                      Reject
                    </Button>
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>

      <RaiseDisputeModal
        open={disputeModalOpen}
        onClose={() => setDisputeModalOpen(false)}
        initialSiteId={site.id}
      />

      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={
          pendingAction?.stage === 'accepted'
            ? 'Accept this site?'
            : pendingAction?.stage === 'rejected'
              ? 'Reject this site?'
              : 'Mark this site inspected?'
        }
        description={
          pendingAction?.stage === 'accepted'
            ? 'This brings the site into service. Nothing else counts as acceptance.'
            : pendingAction?.stage === 'rejected'
              ? 'Sends the site back to Coloplus to rebuild with your reason.'
              : 'Confirms the inspection took place. Next step is accept or reject.'
        }
        tone={pendingAction?.stage === 'rejected' ? 'warning' : 'success'}
        confirmLabel="Confirm"
        pending={transitionMutation.isPending}
        onConfirm={handleConfirmAcceptance}
        onCancel={() => setPendingAction(null)}
      />
    </SidePanel>
  )
}

function ProfileRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
      <dt className="text-sm text-text-muted">{label}</dt>
      <dd className="text-sm font-medium text-text sm:text-right">{value}</dd>
    </div>
  )
}
