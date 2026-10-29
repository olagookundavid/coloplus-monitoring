import type { SVGProps } from 'react'

// No Heroicon depicts a fuel/diesel can (BeakerIcon is a lab flask, only
// loosely related, excluded by frontend/CLAUDE.md §1 rule 2). Used in more
// than one place (Diesel remaining KPI icon, Fuel health card header), so
// it's a shared component.
export function FuelIcon(props: SVGProps<SVGSVGElement>) {
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
      <path d="M5 8a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v11a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8Z" />
      <path d="M9 2.5h4" />
      <path d="M9 6V2.5" />
      <path d="M16 9h1.5a1.5 1.5 0 0 1 1.5 1.5v6a1.5 1.5 0 0 0 3 0V8l-2-2" />
      <path d="M8 12h4" />
    </svg>
  )
}
