import type { ReactNode } from 'react'

import { Button, ButtonLink, SidePanel, StatusBadge } from '../ui'
import { ROUTES } from '../../constants'
import type { MySiteRow } from '../../types'
import { statusLabel } from '../../utils'

export interface SiteProfilePanelProps {
  site: MySiteRow | null
  open: boolean
  onClose: () => void
  onOpenReport: () => void
}

const NOT_AVAILABLE = 'Not available yet'

function Row({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex flex-col gap-1 py-3 sm:flex-row sm:items-center sm:justify-between">
      <dt className="text-sm text-text-muted">{label}</dt>
      <dd className="text-sm font-medium text-text sm:text-right">{value}</dd>
    </div>
  )
}

function accessContactLine(site: MySiteRow): string {
  const { accessContactName: name, accessContactPhone: phone } = site
  if (name && phone) return `${name} · ${phone}`
  return name || phone
}

export function SiteProfilePanel({ site, open, onClose, onOpenReport }: SiteProfilePanelProps) {
  if (!site) return null
  const { label, tone } = statusLabel(site.todayStatus?.status)

  return (
    <SidePanel
      open={open}
      onClose={onClose}
      eyebrow="SITE PROFILE"
      title={`${site.customerSiteCode} · ${site.name}`}
      footer={
        <>
          <ButtonLink variant="outline" to={ROUTES.OPS_INCIDENTS}>
            Log incident
          </ButtonLink>
          <Button
            onClick={() => {
              onClose()
              onOpenReport()
            }}
          >
            Open daily report
          </Button>
        </>
      }
    >
      <dl className="divide-y divide-border">
        <Row label="Type" value={site.siteType || NOT_AVAILABLE} />
        <Row label="Address" value={site.address || NOT_AVAILABLE} />
        <Row label="Generator" value={site.generatorCapacity || NOT_AVAILABLE} />
        <Row label="Access contact" value={accessContactLine(site) || NOT_AVAILABLE} />
        <Row label="Supervisor" value={site.supervisorName || NOT_AVAILABLE} />
        <Row label="Today" value={<StatusBadge label={label} tone={tone} />} />
      </dl>
      <p className="mt-4 text-sm text-text-muted">
        Site details are managed by your supervisor. Contact them if something here is wrong.
      </p>
    </SidePanel>
  )
}
