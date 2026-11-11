import type { AdminSite } from '../../types'
import { Button, Checkbox } from '../ui'

export interface SiteChecklistProps {
  sites: AdminSite[]
  selected: string[]
  onChange: (siteIds: string[]) => void
}

// One shared way to pick a person's sites: invite form and the edit-sites
// dialog both show it. A reviewer or auditor usually needs many sites at
// once (a region, or all), so the list has Select all / Clear.
export function SiteChecklist({ sites, selected, onChange }: SiteChecklistProps) {
  if (sites.length === 0) {
    return (
      <p className="rounded-lg border border-border p-3 text-sm text-text-muted">
        There are no sites yet. Import or add sites first, then assign them here.
      </p>
    )
  }

  function toggle(siteId: string, checked: boolean) {
    onChange(checked ? [...selected, siteId] : selected.filter((id) => id !== siteId))
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-text-muted">
          {selected.length} of {sites.length} selected
        </p>
        <div className="flex items-center gap-1">
          <Button variant="link" size="sm" onClick={() => onChange(sites.map((site) => site.id))}>
            Select all
          </Button>
          <Button variant="link" size="sm" onClick={() => onChange([])}>
            Clear
          </Button>
        </div>
      </div>
      <div className="flex max-h-72 flex-col gap-1 overflow-y-auto rounded-lg border border-border p-2">
        {sites.map((site) => (
          <Checkbox
            key={site.id}
            label={site.name}
            hint={`${site.customerSiteCode} · ${site.state}`}
            checked={selected.includes(site.id)}
            onChange={(event) => toggle(site.id, event.target.checked)}
          />
        ))}
      </div>
    </div>
  )
}
