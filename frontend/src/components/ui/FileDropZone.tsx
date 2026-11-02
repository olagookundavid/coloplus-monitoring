import { PhotoIcon } from '@heroicons/react/24/outline'

export interface FileDropZoneProps {
  label?: string
  hint?: string
  className?: string
}

// A placeholder for evidence/photo upload -- file storage (R2) isn't built
// yet (docs/DECISIONS.md #016, docs/MVP_CHECKLIST.md §11), so this renders
// the drop-zone layout the mockup calls for without any upload wiring.
export function FileDropZone({
  label = 'Drop photos or tickets here',
  hint = 'Evidence upload coming soon.',
  className = '',
}: FileDropZoneProps) {
  return (
    <div
      aria-disabled="true"
      className={`flex flex-col items-center gap-2 rounded-lg border-2 border-dashed border-border bg-surface-subtle px-4 py-8 text-center opacity-60 ${className}`}
    >
      <PhotoIcon className="h-6 w-6 text-text-muted" aria-hidden="true" />
      <p className="text-sm font-medium text-text-muted">{label}</p>
      <p className="text-xs text-text-muted">{hint}</p>
    </div>
  )
}
