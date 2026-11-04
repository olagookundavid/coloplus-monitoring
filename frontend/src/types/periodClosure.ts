export interface PeriodClosure {
  id: string
  /** "YYYY-MM" */
  month: string
  closedBy: string
  closedAt: string
  /** The calculation run frozen at close time. */
  runId: string
  /** `true` means the month IS currently closed/blocked -- the backend's Go
      field is named `Open` internally, but this wire contract and this type
      mean exactly what `locked` says. Don't flip the sense when reading or
      displaying this. */
  locked: boolean
  reopenedBy?: string
  reopenedAt?: string
  reopenReason?: string
}
