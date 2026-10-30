import type { ComponentType, SVGProps } from 'react'

export interface NavItem {
  label: string
  to: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** Checked via usePermissions().can(); absent means "visible to any signed-in user." */
  permission?: string
  badge?: number
}
