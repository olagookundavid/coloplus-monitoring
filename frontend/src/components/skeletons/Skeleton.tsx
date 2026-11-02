import type { CSSProperties } from 'react'

export interface SkeletonProps {
  className?: string
  width?: number | string
  height?: number | string
  rounded?: 'sm' | 'md' | 'lg' | 'full'
}

const ROUNDED_CLASS: Record<NonNullable<SkeletonProps['rounded']>, string> = {
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  full: 'rounded-full',
}

// The one shimmering building block every skeleton in this folder is made
// of. Tailwind's built-in animate-pulse (not MainLoader's stronger, slower
// animate-pulse-fade) is the right amount of motion for routine first-load
// chrome.
export function Skeleton({ className = '', width, height, rounded = 'md' }: SkeletonProps) {
  const style: CSSProperties = {}
  if (width !== undefined) style.width = typeof width === 'number' ? `${width}px` : width
  if (height !== undefined) style.height = typeof height === 'number' ? `${height}px` : height
  return (
    <div
      aria-hidden="true"
      style={style}
      className={`animate-pulse bg-border ${ROUNDED_CLASS[rounded]} ${className}`}
    />
  )
}
