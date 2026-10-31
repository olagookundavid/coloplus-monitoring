import type { ComponentType, SVGProps } from 'react'

export interface NavItem {
  label: string
  to: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** Checked via usePermissions().can(); absent means "visible to any signed-in
      user." A list is "any of" -- any one held permission shows the item. */
  permission?: string | string[]
  badge?: number
}

export interface NavGroup {
  label: string
  items: NavItem[]
}
