import { useMemo, useState, type ReactNode } from 'react'
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import { Link } from 'react-router-dom'
import {
  ArrowRightIcon,
  ChevronUpIcon,
  ChevronDownIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline'

import { Button } from './Button'
import { Input } from './Input'
import { Spinner } from './Spinner'

export interface Column<T> {
  header: string
  accessor: (row: T) => string | number | null
  cell?: (row: T) => ReactNode
  sortable?: boolean
  sortValue?: (row: T) => string | number
  align?: 'left' | 'right' | 'center'
  className?: string
  headerClassName?: string
}

export interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  getRowId: (row: T) => string
  loading?: boolean
  emptyMessage?: string
  searchable?: boolean
  searchPlaceholder?: string
  paginated?: boolean
  pageSize?: number
  headerActions?: ReactNode
  onRowClick?: (row: T) => void
  getRowHref?: (row: T) => string
  className?: string
  rowClassName?: (row: T) => string
}

const ALIGN_CLASS: Record<NonNullable<Column<unknown>['align']>, string> = {
  left: 'text-left',
  right: 'text-right',
  center: 'text-center',
}

export function DataTable<T>({
  data,
  columns,
  getRowId,
  loading = false,
  emptyMessage = 'Nothing to show yet.',
  searchable = true,
  searchPlaceholder = 'Search...',
  paginated = true,
  pageSize = 10,
  headerActions,
  onRowClick,
  getRowHref,
  className = '',
  rowClassName,
}: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize })

  const tableColumns = useMemo<ColumnDef<T, unknown>[]>(
    () =>
      columns.map((column, index) => ({
        id: `col-${index}`,
        accessorFn: (row: T) => column.accessor(row),
        header: column.header,
        cell: (info) =>
          column.cell ? column.cell(info.row.original) : (info.getValue() as ReactNode),
        enableSorting: Boolean(column.sortable),
        sortingFn: column.sortValue
          ? (rowA, rowB) => {
              const a = column.sortValue?.(rowA.original) ?? ''
              const b = column.sortValue?.(rowB.original) ?? ''
              return a < b ? -1 : a > b ? 1 : 0
            }
          : 'auto',
      })),
    [columns]
  )

  const table = useReactTable({
    data,
    columns: tableColumns,
    getRowId,
    state: { sorting, globalFilter, pagination: paginated ? pagination : undefined },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    onPaginationChange: setPagination,
    globalFilterFn: 'includesString',
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: paginated ? getPaginationRowModel() : undefined,
  })

  const rows = table.getRowModel().rows
  const isInteractive = Boolean(onRowClick || getRowHref)

  return (
    <div className={`min-w-0 rounded-lg border border-border bg-surface ${className}`}>
      {(searchable || headerActions) && (
        <div className="flex flex-col gap-3 border-b border-border p-4 sm:flex-row sm:items-center sm:justify-between">
          {searchable && (
            <div className="relative w-full sm:max-w-xs">
              <MagnifyingGlassIcon
                className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted"
                aria-hidden="true"
              />
              <Input
                value={globalFilter}
                onChange={(event) => setGlobalFilter(event.target.value)}
                placeholder={searchPlaceholder}
                className="pl-9"
                aria-label={searchPlaceholder}
              />
            </div>
          )}
          {headerActions && (
            <div className="flex flex-wrap items-center gap-2">{headerActions}</div>
          )}
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="w-full min-w-160 border-collapse text-sm">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id} className="border-b border-border">
                {headerGroup.headers.map((header, columnIndex) => {
                  const column = columns[columnIndex]
                  const align = column?.align ?? 'left'
                  const sorted = header.column.getIsSorted()
                  return (
                    <th
                      key={header.id}
                      scope="col"
                      aria-sort={
                        sorted === 'asc'
                          ? 'ascending'
                          : sorted === 'desc'
                            ? 'descending'
                            : undefined
                      }
                      className={`sticky top-0 bg-surface px-4 py-3 font-medium text-text-muted ${ALIGN_CLASS[align]} ${columnIndex === 0 ? 'sticky left-0 z-10 bg-surface' : ''} ${column?.headerClassName ?? ''}`}
                    >
                      {header.column.getCanSort() ? (
                        <button
                          type="button"
                          onClick={header.column.getToggleSortingHandler()}
                          className="inline-flex cursor-pointer items-center gap-1 font-medium text-text-muted hover:text-text"
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {sorted === 'asc' && (
                            <ChevronUpIcon className="h-3.5 w-3.5" aria-hidden="true" />
                          )}
                          {sorted === 'desc' && (
                            <ChevronDownIcon className="h-3.5 w-3.5" aria-hidden="true" />
                          )}
                        </button>
                      ) : (
                        flexRender(header.column.columnDef.header, header.getContext())
                      )}
                    </th>
                  )
                })}
                {isInteractive && <th scope="col" className="w-11" aria-hidden="true" />}
              </tr>
            ))}
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td
                  colSpan={columns.length + (isInteractive ? 1 : 0)}
                  className="px-4 py-10 text-center text-text-muted"
                >
                  <Spinner className="mx-auto h-5 w-5" />
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length + (isInteractive ? 1 : 0)}
                  className="px-4 py-10 text-center text-text-muted"
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              rows.map((row) => {
                const href = getRowHref?.(row.original)
                const firstColumn = columns[0]
                const rowName =
                  firstColumn && typeof firstColumn.accessor(row.original) === 'string'
                    ? String(firstColumn.accessor(row.original))
                    : undefined
                const detailLabel = rowName ? `View ${rowName}` : 'View details'
                return (
                  <tr
                    key={row.id}
                    className={`border-b border-border last:border-b-0 ${rowClassName?.(row.original) ?? ''}`}
                  >
                    {row.getVisibleCells().map((cell, columnIndex) => {
                      const column = columns[columnIndex]
                      const align = column?.align ?? 'left'
                      const isFirst = columnIndex === 0
                      return (
                        <td
                          key={cell.id}
                          className={`px-4 py-3 align-middle ${ALIGN_CLASS[align]} ${isFirst ? 'sticky left-0 z-10 bg-inherit font-medium text-text' : ''} ${column?.className ?? ''}`}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </td>
                      )
                    })}
                    {isInteractive && (
                      <td className="px-2 text-center align-middle">
                        {href ? (
                          <Link
                            to={href}
                            aria-label={detailLabel}
                            className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                          >
                            <ArrowRightIcon className="h-4 w-4" aria-hidden="true" />
                          </Link>
                        ) : (
                          <button
                            type="button"
                            aria-label={detailLabel}
                            onClick={() => onRowClick?.(row.original)}
                            className="inline-flex h-11 w-11 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface-subtle hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                          >
                            <ArrowRightIcon className="h-4 w-4" aria-hidden="true" />
                          </button>
                        )}
                      </td>
                    )}
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>

      {paginated && table.getPageCount() > 1 && (
        <div className="flex items-center justify-between gap-3 border-t border-border p-4">
          <p className="text-sm text-text-muted">
            Page {pagination.pageIndex + 1} of {table.getPageCount()}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
