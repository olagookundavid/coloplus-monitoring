import { useRef, useState, type ChangeEvent, type DragEvent } from 'react'
import { ArrowUpTrayIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'

import { useUploadAttachmentMutation } from '../../hooks'
import { AuthApiError } from '../../services'
import type { AttachmentOwnerType } from '../../types'
import { ACCEPT_ATTR, toast, validateAttachmentFile } from '../../utils'
import { AlertBanner } from './AlertBanner'
import { Checkbox } from './Checkbox'
import { Spinner } from './Spinner'

export interface FileDropZoneProps {
  ownerType: AttachmentOwnerType
  ownerId: string
  /** Shows the "Visible to Airtel" toggle -- meaningless for an Airtel
      uploader, whose own files are always visible to Coloplus regardless. */
  isStaff: boolean
  label?: string
  hint?: string
  className?: string
}

// A real upload control: request a presigned link, PUT the file straight to
// R2, then confirm (docs/DECISIONS.md #047) -- see services/attachments.ts.
// Uploads as soon as a valid file is picked or dropped, no separate submit
// step.
export function FileDropZone({
  ownerType,
  ownerId,
  isStaff,
  label = 'Drop a file here, or click to browse',
  hint = 'Photos, PDF, text, CSV, Word or Excel. 10 MB max.',
  className = '',
}: FileDropZoneProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const uploadMutation = useUploadAttachmentMutation()
  const [shared, setShared] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  // Sticky, not a toast: storage being off is a system state the person
  // should still see after the toast has faded, not just at the moment of
  // the failed tap.
  const [storageUnavailable, setStorageUnavailable] = useState(false)

  async function handleFile(file: File) {
    const validationError = validateAttachmentFile(file)
    if (validationError) {
      toast.error(validationError)
      return
    }
    try {
      await uploadMutation.mutateAsync({ ownerType, ownerId, file, shared: isStaff ? shared : undefined })
      toast.success('File uploaded.')
    } catch (err) {
      if (err instanceof AuthApiError && err.status === 503) {
        setStorageUnavailable(true)
        return
      }
      toast.error(err instanceof Error ? err.message : 'Could not upload this file.')
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

  const pending = uploadMutation.isPending

  if (storageUnavailable) {
    return (
      <AlertBanner
        tone="warning"
        icon={ExclamationTriangleIcon}
        title="File storage is currently off."
        description="Uploads aren't accepted right now. Try again later."
        className={className}
      />
    )
  }

  return (
    <div className={`flex flex-col gap-2 ${className}`}>
      {isStaff && (
        <Checkbox
          label="Visible to Airtel"
          hint="Off keeps this file internal to Coloplus."
          checked={shared}
          onChange={(event) => setShared(event.target.checked)}
          disabled={pending}
        />
      )}
      <div
        role="button"
        tabIndex={pending ? -1 : 0}
        aria-disabled={pending}
        onClick={() => !pending && inputRef.current?.click()}
        onKeyDown={(event) => {
          if (!pending && (event.key === 'Enter' || event.key === ' ')) {
            event.preventDefault()
            inputRef.current?.click()
          }
        }}
        onDragOver={(event) => {
          event.preventDefault()
          if (!pending) setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={pending ? undefined : handleDrop}
        className={`flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-4 py-8 text-center transition-colors ${
          pending
            ? 'cursor-not-allowed border-border bg-surface-subtle opacity-60'
            : dragOver
              ? 'border-primary bg-primary-soft'
              : 'border-border bg-surface-subtle hover:bg-border/20'
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT_ATTR}
          onChange={handleInputChange}
          disabled={pending}
          className="hidden"
          aria-label={label}
        />
        {pending ? (
          <Spinner className="h-6 w-6" />
        ) : (
          <ArrowUpTrayIcon className="h-6 w-6 text-text-muted" aria-hidden="true" />
        )}
        <p className="text-sm font-medium text-text-muted">
          {pending ? 'Uploading...' : label}
        </p>
        {!pending && <p className="text-xs text-text-muted">{hint}</p>}
      </div>
    </div>
  )
}
