import type { ComponentType, ReactNode, SVGProps } from 'react'
import { ArrowDownTrayIcon } from '@heroicons/react/24/outline'

import { Button } from '../ui'

export interface ReportCardProps {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  title: string
  description: string
  /** An extra field the export needs before it can run (a month or week
      picker) -- rendered between the description and the download button. */
  control?: ReactNode
  /** Disables the download button, e.g. while a required control is empty. */
  disabled?: boolean
  onDownload: () => void
}

export function ReportCard({
  icon: Icon,
  title,
  description,
  control,
  disabled = false,
  onDownload,
}: ReportCardProps) {
  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-lg border border-border bg-surface p-5">
      <span className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-soft text-primary-600">
        <Icon className="h-5 w-5" aria-hidden="true" />
      </span>
      <h2 className="text-base font-semibold text-text">{title}</h2>
      <p className="flex-1 text-sm text-text-muted">{description}</p>
      {control}
      <Button variant="outline" icon={ArrowDownTrayIcon} onClick={onDownload} disabled={disabled}>
        Download CSV
      </Button>
    </div>
  )
}
