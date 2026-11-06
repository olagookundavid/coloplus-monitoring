import { ArrowDownTrayIcon, DocumentIcon } from '@heroicons/react/24/outline'

import { useDownloadAttachment } from '../../hooks'
import type { Attachment } from '../../types'
import { formatDateTimeWAT, formatFileSize, nameOrId } from '../../utils'
import { IconButton } from './IconButton'
import { Spinner } from './Spinner'
import { StatusBadge } from './StatusBadge'

export interface AttachmentListProps {
  attachments: Attachment[]
  loading?: boolean
  emptyMessage?: string
}

export function AttachmentList({
  attachments,
  loading = false,
  emptyMessage = 'No files yet.',
}: AttachmentListProps) {
  const downloadMutation = useDownloadAttachment()

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-text-muted">Files</p>
      {loading ? (
        <Spinner className="mt-3 h-5 w-5" />
      ) : attachments.length === 0 ? (
        <p className="mt-3 text-sm text-text-muted">{emptyMessage}</p>
      ) : (
        <ul className="mt-3 flex flex-col gap-2">
          {attachments.map((attachment) => (
            <li
              key={attachment.id}
              className="flex items-center gap-3 rounded-lg border border-border bg-surface-subtle px-3 py-2.5 text-sm"
            >
              <DocumentIcon className="h-5 w-5 shrink-0 text-text-muted" aria-hidden="true" />
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-text">{attachment.filename}</p>
                <p className="truncate text-xs text-text-muted">
                  {formatFileSize(attachment.sizeBytes)} · {formatDateTimeWAT(attachment.createdAt)}
                  {attachment.uploadedBy &&
                    ` · ${nameOrId(attachment.uploadedByName, attachment.uploadedBy)}`}
                </p>
              </div>
              {attachment.internal && <StatusBadge label="Internal" tone="neutral" />}
              <IconButton
                icon={ArrowDownTrayIcon}
                label={`Download ${attachment.filename}`}
                variant="ghost"
                onClick={() => downloadMutation.mutate(attachment.id)}
                disabled={downloadMutation.isPending}
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
