import { useDataSnapshot } from '../../hooks'

export function DashboardFooter() {
  const snapshot = useDataSnapshot()
  return (
    <footer className="flex flex-col gap-1 border-t border-border px-4 py-4 text-xs text-text-muted sm:flex-row sm:items-center sm:justify-between sm:px-6">
      <p>Coloplus Worldwide Services · Customer Operations</p>
      <p>
        Sample snapshot · {snapshot.dateLabel} · {snapshot.timezoneLabel}
      </p>
    </footer>
  )
}
