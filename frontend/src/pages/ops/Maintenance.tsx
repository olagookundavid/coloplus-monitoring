import { useState } from 'react'
import {
  CheckCircleIcon,
  NoSymbolIcon,
  WrenchScrewdriverIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline'

import { DashboardPageHeader, RequestMaintenanceModal } from '../../components/dashboard'
import {
  Button,
  ConfirmDialog,
  ConfirmSummary,
  DataTable,
  RowActions,
  Select,
  StatusBadge,
  Tabs,
  Textarea,
  type Column,
  type RowAction,
} from '../../components/ui'
import { useAuth } from '../../contexts'
import {
  useAdminSites,
  useApproveMaintenanceMutation,
  useCancelMaintenanceMutation,
  useMaintenanceWindows,
  usePermissions,
  useRejectMaintenanceMutation,
} from '../../hooks'
import type { AdminSite, MaintenanceStatus, MaintenanceWindow } from '../../types'
import { formatDateTimeWAT, maintenanceStatusLabel, nameOrId, toast } from '../../utils'

const NO_WINDOWS: MaintenanceWindow[] = []
const NO_SITES: AdminSite[] = []
const MIN_DECISION_NOTE_LENGTH = 5

type StatusFilter = MaintenanceStatus | 'all'
type PendingAction = { kind: 'approve' | 'reject' | 'cancel'; window: MaintenanceWindow }

export function Maintenance() {
  const { user } = useAuth()
  const { can } = usePermissions()
  const canApprove = can('maintenance.approve')

  const sitesQuery = useAdminSites()
  const sites = sitesQuery.data ?? NO_SITES

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [siteFilter, setSiteFilter] = useState('')

  const windowsQuery = useMaintenanceWindows({
    status: statusFilter === 'all' ? undefined : statusFilter,
    siteId: siteFilter || undefined,
  })
  const windows = windowsQuery.data?.windows ?? NO_WINDOWS
  const rules = windowsQuery.data?.rules

  const [requestOpen, setRequestOpen] = useState(false)
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null)
  const [note, setNote] = useState('')

  const approveMutation = useApproveMaintenanceMutation()
  const rejectMutation = useRejectMaintenanceMutation()
  const cancelMutation = useCancelMaintenanceMutation()
  const mutationPending =
    approveMutation.isPending || rejectMutation.isPending || cancelMutation.isPending

  const trimmedNote = note.trim()
  const noteRequired = pendingAction?.kind !== 'approve'
  const confirmDisabled = noteRequired && trimmedNote.length < MIN_DECISION_NOTE_LENGTH

  function requestAction(kind: PendingAction['kind'], window: MaintenanceWindow) {
    setNote('')
    setPendingAction({ kind, window })
  }

  function handleCancelDialog() {
    setPendingAction(null)
    setNote('')
  }

  function handleConfirm() {
    if (!pendingAction || confirmDisabled) return
    const { kind, window } = pendingAction
    const onSettled = {
      onSuccess: () => {
        toast.success(
          kind === 'approve'
            ? 'Window approved.'
            : kind === 'reject'
              ? 'Window rejected.'
              : 'Window cancelled.'
        )
        handleCancelDialog()
      },
      onError: (error: unknown) => {
        toast.error(error instanceof Error ? error.message : 'Could not update this window.')
        handleCancelDialog()
      },
    }
    if (kind === 'approve') {
      approveMutation.mutate({ id: window.id, note: trimmedNote || undefined }, onSettled)
    } else if (kind === 'reject') {
      rejectMutation.mutate({ id: window.id, note: trimmedNote }, onSettled)
    } else {
      cancelMutation.mutate({ id: window.id, note: trimmedNote }, onSettled)
    }
  }

  const columns: Column<MaintenanceWindow>[] = [
    {
      header: 'Site',
      accessor: (w) => nameOrId(w.siteName, w.siteId),
    },
    {
      header: 'When',
      accessor: (w) => w.startsAt,
      cell: (w) => (
        <div>
          <p className="text-text">{formatDateTimeWAT(w.startsAt)}</p>
          <p className="text-xs text-text-muted">to {formatDateTimeWAT(w.endsAt)}</p>
        </div>
      ),
    },
    { header: 'Minutes', accessor: (w) => w.minutes, align: 'right' },
    {
      header: 'Reason',
      accessor: (w) => w.reason,
      cell: (w) => <span className="line-clamp-2">{w.reason}</span>,
    },
    {
      header: 'Status',
      accessor: (w) => maintenanceStatusLabel(w.status).label,
      cell: (w) => {
        const { label, tone } = maintenanceStatusLabel(w.status)
        return <StatusBadge label={label} tone={tone} />
      },
    },
    {
      header: 'Requested',
      accessor: (w) => w.requestedAt,
      cell: (w) => (
        <div>
          <p className="text-text">{formatDateTimeWAT(w.requestedAt)}</p>
          <p className="text-xs text-text-muted">
            {w.requestedBy ? nameOrId(w.requestedByName, w.requestedBy) : '—'}
          </p>
        </div>
      ),
    },
    {
      header: 'Decision',
      accessor: (w) => w.decidedAt ?? null,
      cell: (w) =>
        w.decidedAt ? (
          <div>
            <p className="text-text">{formatDateTimeWAT(w.decidedAt)}</p>
            <p className="text-xs text-text-muted">
              {w.decidedBy ? nameOrId(w.decidedByName, w.decidedBy) : '—'}
            </p>
            {w.decisionNote && (
              <p className="mt-0.5 text-xs text-text-muted">&ldquo;{w.decisionNote}&rdquo;</p>
            )}
          </div>
        ) : (
          '—'
        ),
    },
    {
      header: 'Actions',
      accessor: () => null,
      align: 'right',
      cell: (w) => {
        const isSelf = Boolean(user && w.requestedBy === user.id)
        const alreadyStarted = new Date(w.startsAt).getTime() <= Date.now()
        const decided = w.status !== 'requested'
        const canCancel = w.status === 'requested' || w.status === 'approved'

        const actions: RowAction[] = []
        if (canApprove) {
          actions.push({
            label: 'Approve',
            icon: CheckCircleIcon,
            disabled: decided || isSelf,
            hint: decided
              ? 'Already decided'
              : isSelf
                ? "You can't approve your own request"
                : undefined,
            onClick: () => requestAction('approve', w),
          })
          actions.push({
            label: 'Reject',
            icon: XCircleIcon,
            disabled: decided || isSelf,
            hint: decided
              ? 'Already decided'
              : isSelf
                ? "You can't decide your own request"
                : undefined,
            onClick: () => requestAction('reject', w),
          })
        }
        if (canApprove || isSelf) {
          actions.push({
            label: 'Cancel',
            icon: NoSymbolIcon,
            disabled: !canCancel || alreadyStarted,
            hint: alreadyStarted
              ? 'This window has already started'
              : !canCancel
                ? 'Already decided'
                : undefined,
            onClick: () => requestAction('cancel', w),
          })
        }

        return <RowActions label="Actions for this window" actions={actions} />
      },
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Maintenance"
        subtitle="Planned power or equipment maintenance windows, excluded from a site's eligible time once approved."
        actions={
          can('maintenance.request') && (
            <Button icon={WrenchScrewdriverIcon} onClick={() => setRequestOpen(true)}>
              Request maintenance
            </Button>
          )
        }
      />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Tabs<StatusFilter>
          aria-label="Filter by status"
          items={[
            { value: 'all', label: 'All' },
            { value: 'requested', label: 'Requested' },
            { value: 'approved', label: 'Approved' },
            { value: 'rejected', label: 'Rejected' },
            { value: 'cancelled', label: 'Cancelled' },
          ]}
          value={statusFilter}
          onChange={setStatusFilter}
        />
        <Select
          value={siteFilter}
          onChange={(event) => setSiteFilter(event.target.value)}
          className="sm:w-56"
          aria-label="Filter by site"
        >
          <option value="">All sites</option>
          {sites.map((site) => (
            <option key={site.id} value={site.id}>
              {site.airtelCode} · {site.name}
            </option>
          ))}
        </Select>
      </div>

      <p className="text-xs text-text-muted">
        Showing the most recent 200 matching windows. Times shown in Africa/Lagos (WAT).
      </p>

      <DataTable
        data={windows}
        columns={columns}
        getRowId={(w) => w.id}
        loading={windowsQuery.isLoading}
        searchable={false}
        emptyMessage="No maintenance windows match this filter."
      />

      <RequestMaintenanceModal
        open={requestOpen}
        onClose={() => setRequestOpen(false)}
        rules={rules}
      />

      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={
          pendingAction?.kind === 'approve'
            ? 'Approve this window?'
            : pendingAction?.kind === 'reject'
              ? 'Reject this window?'
              : 'Cancel this window?'
        }
        description={
          pendingAction?.kind === 'approve'
            ? 'This excludes its minutes from the site’s eligible time.'
            : pendingAction?.kind === 'reject'
              ? 'The requester is notified with your note.'
              : 'The site keeps its full eligible time for this window.'
        }
        body={
          pendingAction && (
            <div className="flex flex-col gap-3">
              <ConfirmSummary
                title={nameOrId(pendingAction.window.siteName, pendingAction.window.siteId)}
                meta={`${formatDateTimeWAT(pendingAction.window.startsAt)} – ${formatDateTimeWAT(pendingAction.window.endsAt)}`}
              />
              <Textarea
                aria-label="Note"
                placeholder={
                  pendingAction.kind === 'approve'
                    ? 'Note (optional)'
                    : 'Say why (at least 5 characters)'
                }
                value={note}
                onChange={(event) => setNote(event.target.value)}
              />
            </div>
          )
        }
        confirmLabel={
          pendingAction?.kind === 'approve'
            ? 'Approve'
            : pendingAction?.kind === 'reject'
              ? 'Reject'
              : 'Cancel window'
        }
        tone={pendingAction?.kind === 'approve' ? 'success' : 'warning'}
        icon={
          pendingAction?.kind === 'approve'
            ? CheckCircleIcon
            : pendingAction?.kind === 'reject'
              ? XCircleIcon
              : NoSymbolIcon
        }
        pending={mutationPending}
        confirmDisabled={confirmDisabled}
        onConfirm={handleConfirm}
        onCancel={handleCancelDialog}
      />
    </div>
  )
}
