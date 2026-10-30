import type { SVGProps } from 'react'

// No Heroicon depicts a telecom site mast/antenna (checked Battery*, RadioIcon,
// SignalIcon, WifiIcon — none fit or are only loosely related). Used in more
// than one place (Sidebar/BottomNav "Sites" nav item, site table cells), so
// it's a shared component per frontend/CLAUDE.md §1 rule 2.
export function SiteMastIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M12 3v18" />
      <path d="M8 7a4 4 0 0 1 8 0" />
      <path d="M5.5 10a6.5 6.5 0 0 1 13 0" />
      <path d="M8.5 21h7" />
      <path d="M12 3l-1.5 3h3L12 3Z" />
    </svg>
  )
}
