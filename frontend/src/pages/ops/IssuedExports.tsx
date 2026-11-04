import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'

import { DashboardPageHeader } from '../../components/dashboard'
import { DataTable, RowActions, type Column } from '../../components/ui'
import { useAuth } from '../../contexts'
import { useDownloadIssuedExportUrl, useIssuedExports } from '../../hooks'
import type { IssuedExport } from '../../types'
import {
  describeIssuedExportParams,
  formatDateTimeWAT,
  formatFileSize,
  issuedByLabel,
  issuedExportKindLabel,
  toast,
} from '../../utils'

const NO_EXPORTS: IssuedExport[] = []

export function IssuedExports() {
  const { user } = useAuth()
  const exportsQuery = useIssuedExports()
  const exports = exportsQuery.data?.exports ?? NO_EXPORTS
  const downloadMutation = useDownloadIssuedExportUrl()

  async function handleDownload(row: IssuedExport) {
    try {
      await downloadMutation.mutateAsync(row.id)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Could not download this export.')
    }
  }

  const columns: Column<IssuedExport>[] = [
    {
      header: 'Export',
      accessor: (row) => row.filename,
      cell: (row) => (
        <div>
          <p className="font-medium text-text">{issuedExportKindLabel(row.kind)}</p>
          <p className="text-xs text-text-muted">{row.filename}</p>
        </div>
      ),
    },
    {
      header: 'Params',
      accessor: (row) => describeIssuedExportParams(row.params),
      cell: (row) => (
        <span className="text-xs text-text-muted">{describeIssuedExportParams(row.params)}</span>
      ),
    },
    { header: 'Rows', accessor: (row) => row.rowCount, align: 'right' },
    {
      header: 'Size',
      accessor: (row) => row.sizeBytes,
      cell: (row) => formatFileSize(row.sizeBytes),
      align: 'right',
    },
    {
      header: 'Issued by',
      accessor: (row) => issuedByLabel(row.issuedBy, user?.id),
    },
    {
      header: 'Issued at',
      accessor: (row) => row.createdAt,
      sortable: true,
      cell: (row) => formatDateTimeWAT(row.createdAt),
    },
    {
      header: 'Actions',
      accessor: () => null,
      align: 'right',
      cell: (row) => (
        <RowActions
          label={`Actions for ${row.filename}`}
          actions={[
            {
              label: 'Download',
              icon: ArrowDownTrayIcon,
              disabled: !row.stored,
              hint: !row.stored ? "The exact file wasn't kept" : undefined,
              onClick: () => void handleDownload(row),
            },
          ]}
        />
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Issued exports"
        subtitle="Every CSV export that's been downloaded, with its fingerprint. An administrator sees every export; everyone else sees only their own."
      />

      <p className="text-xs text-text-muted">
        Showing the most recent 200 issued exports. Times shown in Africa/Lagos (WAT).
      </p>

      <DataTable
        data={exports}
        columns={columns}
        getRowId={(row) => row.id}
        loading={exportsQuery.isLoading}
        searchPlaceholder="Search filename..."
        emptyMessage="No export has been issued yet."
      />
    </div>
  )
}
