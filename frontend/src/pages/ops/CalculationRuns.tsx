import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { CalculatorIcon } from '@heroicons/react/24/outline'

import { CreateCalculationRunModal, DashboardPageHeader } from '../../components/dashboard'
import { SkeletonDetailPane } from '../../components/skeletons'
import {
  AlertBanner,
  Button,
  CalculationDetails,
  DataTable,
  StatusBadge,
  Tabs,
  type Column,
} from '../../components/ui'
import { ROUTES, calculationRunDetailPath } from '../../constants'
import {
  useCalculationRunComparison,
  useCalculationRunDetail,
  useCalculationRuns,
} from '../../hooks'
import type { CalculationRun, CalculationRunRow, RunChange } from '../../types'
import { calculationRunLabel, formatDateTimeWAT, formatRunPercent } from '../../utils'

const NO_RUNS: CalculationRun[] = []
const NO_ROWS: CalculationRunRow[] = []
const NO_CHANGES: RunChange[] = []

type DetailTab = 'rows' | 'compare'

export function CalculationRuns() {
  const { id } = useParams<{ id?: string }>()
  return id ? <RunDetail id={id} /> : <RunList />
}

function RunList() {
  const runsQuery = useCalculationRuns()
  const runs = runsQuery.data?.runs ?? NO_RUNS
  const [createOpen, setCreateOpen] = useState(false)

  const columns: Column<CalculationRun>[] = useMemo(
    () => [
      {
        header: 'Run',
        accessor: (run) => calculationRunLabel(run),
        cell: (run) => (
          <div>
            <p className="font-medium text-text">{calculationRunLabel(run)}</p>
            <p className="text-xs text-text-muted">{run.region || 'All regions'}</p>
          </div>
        ),
      },
      {
        header: 'Period',
        accessor: (run) => `${run.periodFrom} ${run.periodTo}`,
        cell: (run) => `${run.periodFrom} – ${run.periodTo}`,
      },
      { header: 'Formula version', accessor: (run) => `v${run.formulaVersion}` },
      { header: 'Sites', accessor: (run) => run.siteCount, align: 'right' },
      { header: 'Created by', accessor: (run) => run.createdBy },
      {
        header: 'Created at',
        accessor: (run) => run.createdAt,
        sortable: true,
        cell: (run) => formatDateTimeWAT(run.createdAt),
      },
    ],
    []
  )

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Calculation runs"
        subtitle="Figures frozen for a month, kept as an immutable record you can compare against today."
        actions={
          <Button icon={CalculatorIcon} onClick={() => setCreateOpen(true)}>
            New run
          </Button>
        }
      />

      <p className="text-xs text-text-muted">Showing the most recent 200 runs.</p>

      <DataTable
        data={runs}
        columns={columns}
        getRowId={(run) => run.id}
        loading={runsQuery.isLoading}
        getRowHref={(run) => calculationRunDetailPath(run.id)}
        searchPlaceholder="Search runs..."
        emptyMessage="No calculation run has been stored yet."
      />

      <CreateCalculationRunModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  )
}

function RunDetail({ id }: { id: string }) {
  const [tab, setTab] = useState<DetailTab>('rows')
  const detailQuery = useCalculationRunDetail(id)
  const run = detailQuery.data

  if (detailQuery.isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <DashboardPageHeader
          title="Calculation run"
          back={{ to: ROUTES.OPS_CALCULATION_RUNS, label: 'Calculation runs' }}
        />
        <SkeletonDetailPane />
      </div>
    )
  }

  if (!run) {
    return (
      <div className="flex flex-col gap-6">
        <DashboardPageHeader
          title="Calculation run"
          back={{ to: ROUTES.OPS_CALCULATION_RUNS, label: 'Calculation runs' }}
        />
        <AlertBanner tone="danger" title="This run could not be found." />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title={calculationRunLabel(run)}
        subtitle={`${run.periodFrom} – ${run.periodTo} · ${run.region || 'All regions'}`}
        back={{ to: ROUTES.OPS_CALCULATION_RUNS, label: 'Calculation runs' }}
      />

      <CalculationDetails
        formula={run.formula}
        formulaVersion={run.formulaVersion}
        explanation="This run's figures are fixed at the time it was stored. A later correction to an underlying report won't change them here -- compare against today to see what moved."
        rows={[
          { label: 'Period', value: `${run.periodFrom} – ${run.periodTo}` },
          { label: 'Region', value: run.region || 'All regions' },
          { label: 'Sites', value: String(run.siteCount) },
          { label: 'Created by', value: run.createdBy },
          { label: 'Created at', value: formatDateTimeWAT(run.createdAt) },
        ]}
      />

      <Tabs<DetailTab>
        aria-label="Run view"
        items={[
          { value: 'rows', label: 'Run data' },
          { value: 'compare', label: 'Compare to current' },
        ]}
        value={tab}
        onChange={setTab}
      />

      {tab === 'rows' ? <RunRowsTable rows={run.rows ?? NO_ROWS} /> : <RunCompare id={id} />}
    </div>
  )
}

function RunRowsTable({ rows }: { rows: CalculationRunRow[] }) {
  const columns: Column<CalculationRunRow>[] = useMemo(
    () => [
      {
        header: 'Site',
        accessor: (row) => row.siteName,
        cell: (row) => (
          <div>
            <p className="font-medium text-text">{row.siteName}</p>
            <p className="text-xs text-text-muted">{row.airtelCode}</p>
          </div>
        ),
      },
      { header: 'Eligible min', accessor: (row) => row.eligibleMinutes, align: 'right' },
      { header: 'Served min', accessor: (row) => row.servedMinutes, align: 'right' },
      { header: 'Outage min', accessor: (row) => row.outageMinutes, align: 'right' },
      { header: 'Unknown min', accessor: (row) => row.unknownMinutes, align: 'right' },
      { header: 'Assumed min', accessor: (row) => row.assumedMinutes, align: 'right' },
      {
        header: 'Days reported',
        accessor: (row) => `${row.daysReported}/${row.daysInPeriod}`,
        align: 'right',
      },
      {
        header: 'Reported availability',
        accessor: (row) => row.reportedAvailabilityPercent,
        cell: (row) => formatRunPercent(row.reportedAvailabilityPercent),
        align: 'right',
      },
      {
        header: 'Confirmed served share',
        accessor: (row) => row.confirmedServedSharePercent,
        cell: (row) => formatRunPercent(row.confirmedServedSharePercent),
        align: 'right',
      },
      {
        header: 'Coverage',
        accessor: (row) => row.coveragePercent,
        cell: (row) => formatRunPercent(row.coveragePercent),
        align: 'right',
      },
      { header: 'Generator hours', accessor: (row) => row.generatorHours, align: 'right' },
      { header: 'Incidents started', accessor: (row) => row.incidentsStarted, align: 'right' },
    ],
    []
  )

  return (
    <DataTable
      data={rows}
      columns={columns}
      getRowId={(row) => row.siteId}
      searchPlaceholder="Search sites..."
      emptyMessage="No sites in this run."
    />
  )
}

function diffCell(then: number | undefined, now: number | undefined): string {
  if (then === undefined && now === undefined) return '—'
  if (then === undefined) return String(now)
  if (now === undefined) return String(then)
  if (then === now) return String(now)
  return `${then} → ${now}`
}

function RunCompare({ id }: { id: string }) {
  const comparisonQuery = useCalculationRunComparison(id)

  if (comparisonQuery.isLoading) {
    return <SkeletonDetailPane />
  }

  if (comparisonQuery.isError) {
    return (
      <AlertBanner
        tone="danger"
        title="This run can't be compared to today."
        description={
          comparisonQuery.error instanceof Error ? comparisonQuery.error.message : undefined
        }
      />
    )
  }

  const comparison = comparisonQuery.data
  if (!comparison) return null

  const columns: Column<RunChange>[] = [
    {
      header: 'Site',
      accessor: (change) => change.siteName,
      cell: (change) => (
        <div>
          <p className="font-medium text-text">{change.siteName}</p>
          <p className="text-xs text-text-muted">{change.airtelCode}</p>
        </div>
      ),
    },
    {
      header: 'Reports',
      accessor: (change) => (change.reportsChanged ? 'Changed' : 'Same'),
      cell: (change) => (
        <StatusBadge
          label={change.reportsChanged ? 'Changed' : 'Same'}
          tone={change.reportsChanged ? 'warning' : 'neutral'}
        />
      ),
    },
    {
      header: 'Eligible min',
      accessor: () => null,
      align: 'right',
      cell: (change) => diffCell(change.then?.eligibleMinutes, change.now?.eligibleMinutes),
    },
    {
      header: 'Served min',
      accessor: () => null,
      align: 'right',
      cell: (change) => diffCell(change.then?.servedMinutes, change.now?.servedMinutes),
    },
    {
      header: 'Outage min',
      accessor: () => null,
      align: 'right',
      cell: (change) => diffCell(change.then?.outageMinutes, change.now?.outageMinutes),
    },
    {
      header: 'Coverage',
      accessor: () => null,
      align: 'right',
      cell: (change) =>
        diffCell(
          change.then ? Math.round((change.then.coveragePercent ?? 0) * 10) / 10 : undefined,
          change.now ? Math.round((change.now.coveragePercent ?? 0) * 10) / 10 : undefined
        ),
    },
  ]

  return (
    <div className="flex flex-col gap-4">
      <AlertBanner
        tone={comparison.formulaChanged ? 'warning' : 'info'}
        title={
          comparison.formulaChanged
            ? `The formula has changed since this run (now v${comparison.currentFormulaVersion}).`
            : `The formula is unchanged (v${comparison.currentFormulaVersion}).`
        }
      />
      <p className="text-sm text-text-muted">
        {comparison.changes.length} changed · {comparison.unchanged} unchanged · generated{' '}
        {formatDateTimeWAT(comparison.generatedAt)}
      </p>
      <DataTable
        data={comparison.changes ?? NO_CHANGES}
        columns={columns}
        getRowId={(change) => change.siteId}
        searchPlaceholder="Search sites..."
        emptyMessage="Nothing has changed since this run."
      />
    </div>
  )
}
