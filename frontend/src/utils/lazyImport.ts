import { lazy, type ComponentType, type LazyExoticComponent } from 'react'

/**
 * This codebase's pages use named exports, but React.lazy() needs a default
 * export. One small typed helper instead of repeating the .then() adapter
 * at every lazy route. `Record<string, unknown>` (not `any`) accepts any
 * module's shape; the cast to a concrete component type is the one place
 * this trades static proof for the caller supplying the right export name.
 */
export function lazyNamed<P extends object = object>(
  factory: () => Promise<Record<string, unknown>>,
  exportName: string
): LazyExoticComponent<ComponentType<P>> {
  return lazy(async () => {
    const module = await factory()
    return { default: module[exportName] as ComponentType<P> }
  })
}
