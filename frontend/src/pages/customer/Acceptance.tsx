import { useMemo, useState } from 'react'
import { ClipboardDocumentCheckIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { SkeletonDetailPane, SkeletonListItems } from '../../components/skeletons'
import { Button, Spinner, StatusBadge, Tabs, Textarea, type TabItem } from '../../components/ui'
import {
  useAcceptances,
  useAdminSites,
  usePermissions,
  useSiteAcceptance,
  useTransitionAcceptanceMutation,
} from '../../hooks'
import type { Acceptance, AdminSite } from '../../types'
import { acceptanceStageLabel, formatDateTimeWAT, toast } from '../../utils'

const NO_ACCEPTANCES: Acceptance[] = []
const NO_SITES: AdminSite[] = []

type Bucket = 'airtel' | 'coloplus' | 'decided'
type TabValue = Bucket | 'all'

function bucketFor(acceptance: Acceptance): Bucket {
  if (acceptance.stage === 'accepted' || acceptance.stage === 'rejected') return 'decided'
  return acceptance.waitingOn === 'airtel' ? 'airtel' : 'coloplus'
}

function siteLabelFor(sitesById: Map<string, AdminSite>, siteId: string): string {
  const site = sitesById.get(siteId)
  return site ? `${site.name} (${site.airtelCode})` : siteId
}

function AcceptanceDetailPane({
  acceptance,
  siteLabel,
  canAct,
}: {
  acceptance: Acceptance
  siteLabel: string
  canAct: boolean
}) {
  const detailQuery = useSiteAcceptance(acceptance.siteId)
  const events = detailQuery.data?.events ?? []
  const [rejectReason, setRejectReason] = useState('')
  const transitionMutation = useTransitionAcceptanceMutation()

  const { label, tone } = acceptanceStageLabel(acceptance.stage)
  const canActNow = canAct && acceptance.waitingOn === 'airtel'

  async function handleTransition(stage: 'inspected' | 'accepted' | 'rejected', note?: string) {
    try {
      await transitionMutation.mutateAsync({ siteId: acceptance.siteId, input: { stage, note } })
      toast.success(
        stage === 'accepted' ? 'Site accepted.' : stage === 'rejected' ? 'Site rejected.' : 'Marked inspected.'
      )
      setRejectReason('')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not update this acceptance record.')
    }
  }

  return (
    <div className="flex flex-col gap-5 rounded-lg border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge label={label} tone={tone} />
        {acceptance.overdue && <StatusBadge label="Overdue" tone="danger" />}
      </div>

      <h2 className="text-xl font-bold text-text">{siteLabel}</h2>

      <dl className="divide-y divide-border rounded-lg border border-border">
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Waiting on</dt>
          <dd className="font-medium text-text">
            {acceptance.waitingOn === 'airtel' ? 'Airtel' : 'Coloplus'}
          </dd>
        </div>
        {acceptance.nextAction && (
          <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
            <dt className="text-text-muted">Next action</dt>
            <dd className="font-medium text-text">{acceptance.nextAction}</dd>
          </div>
        )}
        <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
          <dt className="text-text-muted">Ready for inspection</dt>
          <dd className="font-medium text-text">{acceptance.readyAgeHours}h ago</dd>
        </div>
        {acceptance.dossier && (
          <div className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
            <dt className="text-text-muted">Dossier</dt>
            <dd className="max-w-[60%] text-right font-medium text-text">{acceptance.dossier}</dd>
          </div>
        )}
      </dl>

      <div>
        <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Timeline</p>
        {detailQuery.isLoading ? (
          <Spinner className="mt-3 h-5 w-5" />
        ) : events.length === 0 ? (
          <p className="mt-3 text-sm text-text-muted">No activity yet.</p>
        ) : (
          <ul className="mt-3 flex flex-col gap-3">
            {events.map((event) => (
              <li key={event.id} className="flex gap-3 text-sm">
                <span className="w-32 shrink-0 text-text-muted">
                  {formatDateTimeWAT(event.createdAt)}
                </span>
                <span className="text-text">{event.body || `Moved to ${event.toStage}`}</span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {canActNow && acceptance.stage === 'ready_for_inspection' && (
        <Button onClick={() => handleTransition('inspected')}>Mark inspected</Button>
      )}

      {canActNow && acceptance.stage === 'inspected' && (
        <div className="flex flex-col gap-3">
          <Button onClick={() => handleTransition('accepted')} fullWidth>
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
            onClick={() => handleTransition('rejected', rejectReason.trim())}
            fullWidth
          >
            Reject
          </Button>
        </div>
      )}
    </div>
  )
}

export function Acceptance() {
  const { can } = usePermissions()
  const canReview = can('acceptance.review')

  const acceptancesQuery = useAcceptances()
  const acceptances = acceptancesQuery.data ?? NO_ACCEPTANCES
  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES
  const sitesById = useMemo(() => new Map(sites.map((site) => [site.id, site])), [sites])

  const [tab, setTab] = useState<TabValue>('airtel')
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)

  const counts = useMemo(() => {
    const result = { airtel: 0, coloplus: 0, decided: 0 }
    for (const acceptance of acceptances) result[bucketFor(acceptance)]++
    return result
  }, [acceptances])

  const visible = useMemo(
    () => (tab === 'all' ? acceptances : acceptances.filter((a) => bucketFor(a) === tab)),
    [acceptances, tab]
  )

  const selected = visible.find((a) => a.siteId === selectedSiteId) ?? visible[0] ?? null

  const tabItems: TabItem<TabValue>[] = [
    { value: 'airtel', label: `Waiting on you ${counts.airtel}` },
    { value: 'coloplus', label: `With Coloplus ${counts.coloplus}` },
    { value: 'decided', label: `Decided ${counts.decided}` },
    { value: 'all', label: `All ${acceptances.length}` },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        eyebrow="Commissioning"
        title="Acceptance"
        subtitle="Sites waiting to go into service. Acceptance is final — nothing else counts as it."
      />

      <Tabs aria-label="Filter" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {acceptancesQuery.isLoading ? (
              <SkeletonListItems count={5} />
            ) : visible.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                Nothing here.
              </li>
            ) : (
              visible.map((acceptance) => {
                const sameSite = acceptance.siteId === (selected?.siteId ?? null)
                const { label, tone } = acceptanceStageLabel(acceptance.stage)
                return (
                  <li key={acceptance.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedSiteId(acceptance.siteId)}
                      className={`w-full min-w-0 rounded-lg border p-4 text-left transition-colors ${
                        sameSite
                          ? 'border-primary bg-primary-soft'
                          : 'border-border bg-surface hover:bg-surface-subtle'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <StatusBadge label={label} tone={tone} />
                        {acceptance.overdue && <StatusBadge label="Overdue" tone="danger" />}
                      </div>
                      <p className="mt-2 truncate font-medium text-text">
                        {siteLabelFor(sitesById, acceptance.siteId)}
                      </p>
                    </button>
                  </li>
                )
              })
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {acceptancesQuery.isLoading ? (
            <SkeletonDetailPane />
          ) : !selected ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              <ClipboardDocumentCheckIcon className="mb-2 h-6 w-6 text-text-muted" aria-hidden="true" />
              Select a site to see its acceptance record.
            </div>
          ) : (
            <AcceptanceDetailPane
              key={selected.siteId}
              acceptance={selected}
              siteLabel={siteLabelFor(sitesById, selected.siteId)}
              canAct={canReview}
            />
          )}
        </div>
      </div>
    </div>
  )
}
