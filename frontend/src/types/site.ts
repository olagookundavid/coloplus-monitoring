export type SiteStatus = 'Available' | 'Low diesel' | 'Power outage' | 'Update overdue'
export type PowerSource = 'Grid' | 'Generator' | 'Battery' | 'No supply' | 'Awaiting update'

export interface Site {
  id: string
  code: string
  name: string
  region: string
  status: SiteStatus
  powerSource: PowerSource
  energyKwh: number
  dieselPercent: number
  dieselTankCapacityLitres: number
  powerAvailabilityPercent: number
  /** null = not reported — never 0, never the last-known value repeated. */
  radioAvailabilityPercent: number | null
  /** ISO 8601. */
  updatedAt: string
}
