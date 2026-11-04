import { useState } from 'react'
import { Link } from 'react-router-dom'
import { LockClosedIcon, LockOpenIcon } from '@heroicons/react/24/outline'

import { ClosePeriodModal, DashboardPageHeader } from '../../components/dashboard'
import {
  Button,
  ConfirmDialog,
  ConfirmSummary,
  DataTable,
  RowActions,
  StatusBadge,
  Textarea,
  type Column,
} from '../../components/ui'
import { calculationRunDetailPath } from '../../constants'
import { useReopenPeriodMutation, usePeriods } from '../../hooks'
import type { PeriodClosure } from '../../types'
import { formatDateTimeWAT, periodStatusLabel, toast } from '../../utils'

const NO_PERIODS: PeriodClosure[] = []
const MIN_REOPEN_REASON_LENGTH = 10

export function Periods() {
  const periodsQuery = usePeriods()
  const periods = periodsQuery.data ?? NO_PERIODS

  const [closeOpen, setCloseOpen] = useState(false)
  const [pendingPeriod, setPendingPeriod] = useState<PeriodClosure | null>(null)
  const [reason, setReason] = useState('')

  const reopenMutation = useReopenPeriodMutation()
  const trimmedReason = reason.trim()

  function requestReopen(period: PeriodClosure) {
    setReason('')
    setPendingPeriod(period)
  }

  function handleCancel() {
    setPendingPeriod(null)
    setReason('')
  }

  function handleConfirm() {
    if (!pendingPeriod || trimmedReason.length < MIN_REOPEN_REASON_LENGTH) return
    reopenMutation.mutate(
      { month: pendingPeriod.month, reason: trimmedReason },
      {
        onSuccess: () => {
          toast.success(`${pendingPeriod.month} is reopened.`)
          handleCancel()
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Could not reopen this month.')
          handleCancel()
        },
      }
    )
  }

  const columns: Column<PeriodClosure>[] = [
    { header: 'Month', accessor: (period) => period.month },
    {
      header: 'Status',
      accessor: (period) => (period.locked ? 'Closed' : 'Reopened'),
      cell: (period) => {
        const { label, tone } = periodStatusLabel(period.locked)
        return <StatusBadge label={label} tone={tone} />
      },
    },
    {
      header: 'Closed',
      accessor: (period) => period.closedAt,
      cell: (period) => (
        <div>
          <p className="text-text">{formatDateTimeWAT(period.closedAt)}</p>
          <p className="text-xs text-text-muted">{period.closedBy}</p>
        </div>
      ),
    },
    {
      header: 'Run',
      accessor: (period) => period.runId,
      cell: (period) => (
        <Link to={calculationRunDetailPath(period.runId)} className="text-primary hover:underline">
          View run
        </Link>
      ),
    },
    {
      header: 'Reopened',
      accessor: (period) => period.reopenedAt ?? null,
      cell: (period) =>
        period.reopenedAt ? (
          <div>
            <p className="text-text">{formatDateTimeWAT(period.reopenedAt)}</p>
            <p className="text-xs text-text-muted">{period.reopenedBy}</p>
            {period.reopenReason && (
              <p className="mt-0.5 text-xs text-text-muted">&ldquo;{period.reopenReason}&rdquo;</p>
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
      cell: (period) => (
        <RowActions
          label={`Actions for ${period.month}`}
          actions={[
            {
              label: 'Reopen',
              icon: LockOpenIcon,
              disabled: !period.locked,
              hint: !period.locked ? "This month isn't currently closed" : undefined,
              onClick: () => requestReopen(period),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Closing periods"
        subtitle="Close a month to lock its reports and freeze that month's figures. Reopening needs a reason."
        actions={
          <Button icon={LockClosedIcon} onClick={() => setCloseOpen(true)}>
            Close a month
          </Button>
        }
      />

      <DataTable
        data={periods}
        columns={columns}
        getRowId={(period) => period.id}
        loading={periodsQuery.isLoading}
        searchable={false}
        emptyMessage="No month has been closed yet."
      />

      <ClosePeriodModal open={closeOpen} onClose={() => setCloseOpen(false)} />

      <ConfirmDialog
        open={Boolean(pendingPeriod)}
        title="Reopen this month?"
        description="Reports for this month can be added, corrected and approved again until it's closed once more."
        body={
          pendingPeriod && (
            <div className="flex flex-col gap-3">
              <ConfirmSummary
                title={pendingPeriod.month}
                meta={`Closed ${formatDateTimeWAT(pendingPeriod.closedAt)} by ${pendingPeriod.closedBy}`}
              />
              <Textarea
                aria-label="Reason for reopening"
                placeholder="Say why this month is being reopened (at least 10 characters)"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
            </div>
          )
        }
        confirmLabel="Reopen"
        tone="warning"
        icon={LockOpenIcon}
        pending={reopenMutation.isPending}
        confirmDisabled={trimmedReason.length < MIN_REOPEN_REASON_LENGTH}
        onConfirm={handleConfirm}
        onCancel={handleCancel}
      />
    </div>
  )
}
