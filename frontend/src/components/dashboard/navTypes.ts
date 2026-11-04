import type { ComponentType, SVGProps } from 'react'

export interface NavItem {
  label: string
  to: string
  icon: ComponentType<SVGProps<SVGSVGElement>>
  /** Checked via usePermissions().can(); absent means "visible to any signed-in
      user." A list is "any of" -- any one held permission shows the item. */
  permission?: string | string[]
  /** Hides an otherwise-visible item from anyone who ALSO holds one of these
      permissions -- for a role whose permissions are a subset of one or more
      other roles' (auditor's five permissions are each also held by admin,
      and some by reporter/supervisor too), so those other roles' sidebars
      don't show a confusing duplicate section. Same "any of" semantics as
      `permission`. */
  excludePermission?: string | string[]
  badge?: number
}

export interface NavGroup {
  label: string
  items: NavItem[]
}
