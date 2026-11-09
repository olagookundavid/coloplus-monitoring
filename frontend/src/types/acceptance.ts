// Mirrors dto.AcceptanceDTO (backend/internal/dto/acceptance.go). Unlike
// disputes/tickets, this DTO carries no per-person identity fields, so it's
// the same shape for every caller -- no staff-only fields to hide.

// Mirrors domain.AcceptanceStage.
export type AcceptanceStage =
  'requested' | 'built' | 'ready_for_inspection' | 'inspected' | 'accepted' | 'rejected' | 'on_hold'

export type AcceptanceParty = 'customer' | 'coloplus'

export interface AcceptanceEvent {
  id: string
  party: AcceptanceParty
  fromStage?: AcceptanceStage
  toStage: AcceptanceStage
  body: string
  createdAt: string
}

export interface Acceptance {
  id: string
  siteId: string
  siteName?: string
  stage: AcceptanceStage
  /** The stage a hold returns to. */
  heldFrom?: AcceptanceStage
  /** Whose move it is. */
  waitingOn: AcceptanceParty
  nextAction?: string
  dueAt?: string
  overdue: boolean
  /** What Coloplus submitted when it declared the site ready. */
  dossier?: string
  reason?: string
  holdReason?: string
  /** How long the site has waited for inspection since being declared ready -- not operational performance. */
  readyAgeHours: number
  requestedAt: string
  builtAt?: string
  readyAt?: string
  inspectedAt?: string
  /** When the customer accepted. Nothing else counts as acceptance. */
  decidedAt?: string
  updatedAt: string
}

export interface AcceptanceDetail {
  acceptance: Acceptance
  events: AcceptanceEvent[]
}

export interface TransitionAcceptanceInput {
  /** inspected, accepted, rejected, on_hold, or the held stage to resume. */
  stage: AcceptanceStage
  note?: string
}
