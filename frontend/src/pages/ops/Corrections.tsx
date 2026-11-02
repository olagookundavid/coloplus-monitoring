import { useMemo, useState } from 'react'
import { PencilSquareIcon, PlusIcon } from '@heroicons/react/24/outline'

import {
  CorrectionDetailPane,
  CorrectionListCard,
  DailyReportPanel,
  DashboardPageHeader,
  RequestCorrectionModal,
} from '../../components/dashboard'
import type {
  PendingCorrectionRequest,
  RequestCorrectionModalInitial,
} from '../../components/dashboard'
import { SkeletonDetailPane, SkeletonListItems } from '../../components/skeletons'
import { Button, ConfirmDialog, ConfirmSummary, Tabs, type TabItem } from '../../components/ui'
import {
  useCorrections,
  useRequestIncidentCorrectionMutation,
  useRequestReportCorrectionMutation,
} from '../../hooks'
import type {
  CorrectionListItem,
  CorrectionTab,
  DailyReport,
  IncidentCorrection,
  SiteReportRow,
} from '../../types'
import { correctionTabInfo, toast } from '../../utils'

interface PanelTarget {
  site: SiteReportRow
  reportId: string
  reportDate: string
}

export function Corrections() {
  const { items, sites, sitesById, isLoading } = useCorrections()

  const [tab, setTab] = useState<CorrectionTab>('needsAction')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [panelTarget, setPanelTarget] = useState<PanelTarget | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [requestAgainInitial, setRequestAgainInitial] =
    useState<RequestCorrectionModalInitial | null>(null)
  const [pendingRequest, setPendingRequest] = useState<PendingCorrectionRequest | null>(null)

  const reportCorrectionMutation = useRequestReportCorrectionMutation()
  const incidentCorrectionMutation = useRequestIncidentCorrectionMutation()
  const requestPending = reportCorrectionMutation.isPending || incidentCorrectionMutation.isPending

  const counts = useMemo(() => {
    const result = { needsAction: 0, awaitingReview: 0, approved: 0 }
    for (const item of items) result[item.tab]++
    return result
  }, [items])

  const visibleItems = useMemo(() => items.filter((item) => item.tab === tab), [items, tab])
  const selectedItem: CorrectionListItem | null =
    visibleItems.find((item) => item.id === selectedId) ?? visibleItems[0] ?? null

  const tabItems: TabItem<CorrectionTab>[] = [
    {
      value: 'needsAction',
      label: `${correctionTabInfo('needsAction').label} ${counts.needsAction}`,
    },
    {
      value: 'awaitingReview',
      label: `${correctionTabInfo('awaitingReview').label} ${counts.awaitingReview}`,
    },
    { value: 'approved', label: `${correctionTabInfo('approved').label} ${counts.approved}` },
  ]

  function openPanel(report: DailyReport) {
    const site = sitesById.get(report.siteId) ?? {
      siteId: report.siteId,
      airtelCode: report.siteId,
      coloplusCode: report.siteId,
      name: report.siteId,
      siteType: '',
      state: '',
    }
    setPanelTarget({ site, reportId: report.id, reportDate: report.reportDate })
  }

  function openRequestAgain(correction: IncidentCorrection) {
    setRequestAgainInitial({
      incidentId: correction.incidentId,
      siteId: correction.siteId,
      field: correction.field,
    })
    setCreateOpen(true)
  }

  function closeCreateModal() {
    setCreateOpen(false)
    setRequestAgainInitial(null)
  }

  async function handleConfirmRequest() {
    if (!pendingRequest) return
    try {
      if (pendingRequest.target === 'report') {
        await reportCorrectionMutation.mutateAsync({
          reportId: pendingRequest.reportId,
          field: pendingRequest.field,
          correctedValue: pendingRequest.correctedValue,
          reason: pendingRequest.reason,
        })
      } else {
        await incidentCorrectionMutation.mutateAsync({
          incidentId: pendingRequest.incidentId,
          field: pendingRequest.field,
          correctedValue: pendingRequest.correctedValue,
          reason: pendingRequest.reason,
        })
      }
      toast.success('Correction request sent.')
      setPendingRequest(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not send this correction request.')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Corrections"
        subtitle="Fix reports your supervisor returned, or request a change to a record that is already submitted. Originals are always kept."
        actions={
          <Button icon={PlusIcon} onClick={() => setCreateOpen(true)}>
            Request a correction
          </Button>
        }
      />

      <Tabs aria-label="Filter by status" items={tabItems} value={tab} onChange={setTab} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <div className="flex min-w-0 flex-col gap-3 lg:col-span-2">
          <ul className="flex flex-col gap-2">
            {isLoading ? (
              <SkeletonListItems count={5} />
            ) : visibleItems.length === 0 ? (
              <li className="rounded-lg border border-border bg-surface p-4 text-sm text-text-muted">
                {tab === 'needsAction'
                  ? "Nothing waiting on you -- every report you've filed is in good standing."
                  : 'Nothing here.'}
              </li>
            ) : (
              visibleItems.map((item) => (
                <li key={item.id}>
                  <CorrectionListCard
                    item={item}
                    site={sitesById.get(item.siteId)}
                    selected={item.id === (selectedItem?.id ?? null)}
                    onSelect={() => setSelectedId(item.id)}
                  />
                </li>
              ))
            )}
          </ul>
        </div>

        <div className="min-w-0 lg:col-span-3">
          {isLoading ? (
            <SkeletonDetailPane />
          ) : !selectedItem ? (
            <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-border p-8 text-center text-sm text-text-muted">
              Select a record to see its details.
            </div>
          ) : (
            <CorrectionDetailPane
              key={selectedItem.id}
              item={selectedItem}
              site={sitesById.get(selectedItem.siteId)}
              onFixReport={openPanel}
              onRequestAgain={openRequestAgain}
            />
          )}
        </div>
      </div>

      {panelTarget && (
        <DailyReportPanel
          open={Boolean(panelTarget)}
          onClose={() => setPanelTarget(null)}
          site={panelTarget.site}
          reportDate={panelTarget.reportDate}
          reportId={panelTarget.reportId}
          mode="edit"
        />
      )}

      <RequestCorrectionModal
        open={createOpen}
        onClose={closeCreateModal}
        sites={sites}
        initial={requestAgainInitial}
        onRequestConfirm={setPendingRequest}
      />

      <ConfirmDialog
        open={Boolean(pendingRequest)}
        title="Send this correction request?"
        description="Your supervisor reviews it before anything changes. The original stays on file either way."
        tone="info"
        icon={PencilSquareIcon}
        confirmLabel="Send request"
        pending={requestPending}
        body={
          pendingRequest && (
            <ConfirmSummary title={pendingRequest.siteLabel} meta={pendingRequest.fieldLabel} />
          )
        }
        onConfirm={handleConfirmRequest}
        onCancel={() => setPendingRequest(null)}
      />
    </div>
  )
}
