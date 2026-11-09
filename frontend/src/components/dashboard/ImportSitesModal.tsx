import { useRef, useState, type ChangeEvent, type DragEvent } from 'react'
import { DocumentArrowUpIcon } from '@heroicons/react/24/outline'

import { useImportSitesMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import type { SiteImportRow } from '../../types'
import { siteImportRowActionLabel, toast, validateSiteImportFile } from '../../utils'
import { AlertBanner, Button, DataTable, Modal, Spinner, StatusBadge, type Column } from '../ui'

export interface ImportSitesModalProps {
  open: boolean
  onClose: () => void
}

const NO_ROWS: SiteImportRow[] = []

const COLUMNS: Column<SiteImportRow>[] = [
  { header: 'Line', accessor: (row) => row.line },
  { header: 'Customer site code', accessor: (row) => row.customerSiteCode },
  {
    header: 'Action',
    accessor: (row) => row.action,
    cell: (row) => {
      const { label, tone } = siteImportRowActionLabel(row.action)
      return <StatusBadge label={label} tone={tone} />
    },
  },
  {
    header: 'Problems',
    accessor: (row) => row.problems?.join('; ') ?? null,
    cell: (row) =>
      row.problems && row.problems.length > 0 ? (
        <ul className="list-inside list-disc">
          {row.problems.map((problem, index) => (
            <li key={index}>{problem}</li>
          ))}
        </ul>
      ) : (
        '—'
      ),
  },
]

// Two internal stages: pick a file (dry-run it immediately) -> preview the
// result, with its per-row actions/problems *as* the confirmation -- there
// is no separate ConfirmDialog on top of "Import N sites". Matched by
// Customer site code, all-or-nothing: see backend/internal/services/site_import.go.
export function ImportSitesModal({ open, onClose }: ImportSitesModalProps) {
  const mutation = useImportSitesMutation()
  const inputRef = useRef<HTMLInputElement>(null)

  const [csvText, setCsvText] = useState<string | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const [rows, setRows] = useState<SiteImportRow[] | null>(null)
  const [stats, setStats] = useState<{
    created: number
    updated: number
    unchanged: number
    rejected: number
  } | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const checking = mutation.isPending && !rows
  const importing = mutation.isPending && Boolean(rows)

  function reset() {
    setCsvText(null)
    setFileName(null)
    setRows(null)
    setStats(null)
    setFormError(null)
  }

  function handleClose() {
    onClose()
    reset()
  }

  async function handleFile(file: File) {
    const validationError = validateSiteImportFile(file)
    if (validationError) {
      toast.error(validationError)
      return
    }
    setFormError(null)
    const text = await file.text()
    setFileName(file.name)
    try {
      const dryRunResult = await mutation.mutateAsync({ csv: text, dryRun: true })
      setCsvText(text)
      setRows(dryRunResult.rows)
      setStats({
        created: dryRunResult.created,
        updated: dryRunResult.updated,
        unchanged: dryRunResult.unchanged,
        rejected: dryRunResult.rejected,
      })
    } catch (error) {
      setFormError(error instanceof AuthApiError ? error.message : 'Could not read this file.')
    }
  }

  function handleInputChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (file) void handleFile(file)
  }

  function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault()
    setDragOver(false)
    const file = event.dataTransfer.files?.[0]
    if (file) void handleFile(file)
  }

  async function handleImport() {
    if (!csvText) return
    setFormError(null)
    try {
      const applied = await mutation.mutateAsync({ csv: csvText, dryRun: false })
      toast.success(`Imported ${applied.created + applied.updated} sites.`)
      handleClose()
    } catch (error) {
      setFormError(error instanceof AuthApiError ? error.message : 'Could not import this file.')
    }
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Import sites from CSV"
      subtitle="Matched by Customer site code. A blank cell on an existing site leaves that field unchanged."
      maxWidth="3xl"
      footer={
        stats ? (
          <>
            <Button variant="outline" onClick={reset} disabled={importing}>
              Choose a different file
            </Button>
            <Button onClick={handleImport} loading={importing} disabled={stats.rejected > 0}>
              Import {stats.created + stats.updated} sites
            </Button>
          </>
        ) : (
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
        )
      }
    >
      <div className="flex flex-col gap-4">
        {formError && (
          <div
            role="alert"
            className="rounded-lg border border-danger bg-danger-50 p-3 text-sm text-danger-600"
          >
            {formError}
          </div>
        )}

        {!stats && (
          <div
            role="button"
            tabIndex={checking ? -1 : 0}
            aria-disabled={checking}
            onClick={() => !checking && inputRef.current?.click()}
            onKeyDown={(event) => {
              if (!checking && (event.key === 'Enter' || event.key === ' ')) {
                event.preventDefault()
                inputRef.current?.click()
              }
            }}
            onDragOver={(event) => {
              event.preventDefault()
              if (!checking) setDragOver(true)
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={checking ? undefined : handleDrop}
            className={`flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-4 py-8 text-center transition-colors ${
              checking
                ? 'cursor-not-allowed border-border bg-surface-subtle opacity-60'
                : dragOver
                  ? 'border-primary bg-primary-soft'
                  : 'border-border bg-surface-subtle hover:bg-border/20'
            }`}
          >
            <input
              ref={inputRef}
              type="file"
              accept=".csv,text/csv"
              onChange={handleInputChange}
              disabled={checking}
              className="hidden"
              aria-label="Choose a CSV file"
            />
            {checking ? (
              <Spinner className="h-6 w-6" />
            ) : (
              <DocumentArrowUpIcon className="h-6 w-6 text-text-muted" aria-hidden="true" />
            )}
            <p className="text-sm font-medium text-text-muted">
              {checking ? 'Checking the file...' : 'Drop a CSV file here, or click to browse'}
            </p>
            {!checking && (
              <p className="text-xs text-text-muted">
                Header row with customer_site_code and any other site column. Up to 1,000 rows, 800 KB.
              </p>
            )}
          </div>
        )}

        {stats && (
          <>
            <p className="text-sm text-text-muted">
              {fileName} · {(rows ?? NO_ROWS).length} row
              {(rows ?? NO_ROWS).length === 1 ? '' : 's'}
            </p>

            <div className="flex flex-wrap gap-4 text-sm text-text-muted">
              <span>
                Create <strong className="text-text">{stats.created}</strong>
              </span>
              <span>
                Update <strong className="text-text">{stats.updated}</strong>
              </span>
              <span>
                Unchanged <strong className="text-text">{stats.unchanged}</strong>
              </span>
              <span>
                Rejected <strong className="text-text">{stats.rejected}</strong>
              </span>
            </div>

            {stats.rejected > 0 && (
              <AlertBanner
                tone="danger"
                title="Nothing will be written."
                description="Fix every row with a problem and choose the file again — the import is all-or-nothing."
              />
            )}

            <DataTable
              data={rows ?? NO_ROWS}
              columns={COLUMNS}
              getRowId={(row) => String(row.line)}
              compact
              searchable={false}
              paginated={(rows ?? NO_ROWS).length > 10}
              emptyMessage="No rows."
            />
          </>
        )}
      </div>
    </Modal>
  )
}
