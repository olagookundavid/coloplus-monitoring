package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type AcceptanceStage string

const (
	StageRequested          AcceptanceStage = "requested"
	StageBuilt              AcceptanceStage = "built"
	StageReadyForInspection AcceptanceStage = "ready_for_inspection"
	StageInspected          AcceptanceStage = "inspected"
	StageAccepted           AcceptanceStage = "accepted"
	StageRejected           AcceptanceStage = "rejected"
	StageOnHold             AcceptanceStage = "on_hold"
)

func ValidAcceptanceStage(s AcceptanceStage) bool {
	switch s {
	case StageRequested, StageBuilt, StageReadyForInspection, StageInspected, StageAccepted,
		StageRejected, StageOnHold:
		return true
	}
	return false
}

const maxAcceptanceText = 4000

// Acceptance is a site's commissioning and acceptance process. It is kept
// apart from the operational record: reporting well never counts as Airtel's
// acceptance, and nothing but Airtel's own decision does.
type Acceptance struct {
	ID     string
	SiteID string
	Stage  AcceptanceStage
	// HeldFrom is the stage a hold returns to.
	HeldFrom    AcceptanceStage
	Responsible DisputeParty
	NextAction  string
	DueAt       *time.Time
	// Dossier is what Coloplus submitted when it declared the site ready.
	Dossier string
	// Reason is why it was rejected; HoldReason is why it is on hold.
	Reason      string
	HoldReason  string
	RequestedAt time.Time
	BuiltAt     *time.Time
	ReadyAt     *time.Time
	InspectedAt *time.Time
	DecidedAt   *time.Time
	DecidedBy   string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Done reports whether the process has ended in acceptance.
func (a Acceptance) Done() bool { return a.Stage == StageAccepted }

// Overdue reports whether the next action is late. A hold stops the clock and
// an accepted site has nothing left to do.
func (a Acceptance) Overdue(now time.Time) bool {
	return a.DueAt != nil && a.DueAt.Before(now) && a.Stage != StageAccepted && a.Stage != StageOnHold
}

// ReadyAge is how long the site has waited for inspection since Coloplus
// declared it ready: zero when it is not waiting. It is reported apart from
// operational performance.
func (a Acceptance) ReadyAge(now time.Time) time.Duration {
	if a.ReadyAt == nil || (a.Stage != StageReadyForInspection && a.Stage != StageInspected) {
		return 0
	}
	return now.Sub(*a.ReadyAt)
}

// AcceptanceStep is one move along the process.
type AcceptanceStep struct {
	To   AcceptanceStage
	Note string
	// NextAction and DueAt are the plan after this step.
	NextAction string
	DueAt      *time.Time
}

// CanMove says whether the party may take the process from where it is to the
// step's stage. Coloplus builds and submits; only Airtel inspects, accepts or
// rejects. Either side may hold or resume. Accepted is final.
func (a Acceptance) CanMove(party DisputeParty, to AcceptanceStage) bool {
	if a.Stage == StageAccepted {
		return false
	}
	if a.Stage == StageOnHold {
		return to == a.HeldFrom
	}
	if to == StageOnHold {
		return true
	}
	switch party {
	case PartyColoplus:
		switch a.Stage {
		case StageRequested:
			return to == StageBuilt
		case StageBuilt:
			return to == StageReadyForInspection
		case StageRejected:
			return to == StageBuilt
		}
	case PartyAirtel:
		switch a.Stage {
		case StageReadyForInspection:
			return to == StageInspected || to == StageRejected
		case StageInspected:
			return to == StageAccepted || to == StageRejected
		}
	}
	return false
}

// CheckStep lists what is missing in the step, apart from whether the move is
// allowed.
func (a Acceptance) CheckStep(party DisputeParty, s AcceptanceStep, now time.Time) []string {
	var problems []string
	note := strings.TrimSpace(s.Note)
	if utf8.RuneCountInString(note) > maxAcceptanceText || utf8.RuneCountInString(s.NextAction) > maxAcceptanceText {
		problems = append(problems, "the note is too long")
	}
	if s.DueAt != nil && !s.DueAt.After(now) {
		problems = append(problems, "the due date must be in the future")
	}
	switch s.To {
	case StageReadyForInspection:
		if note == "" {
			problems = append(problems, "submit the dossier: say what has been built and tested, and where the evidence is")
		}
	case StageRejected:
		if note == "" {
			problems = append(problems, "give the reason for rejecting")
		}
	case StageOnHold:
		if note == "" {
			problems = append(problems, "give the reason for the hold")
		}
	case StageBuilt:
		if a.Stage == StageRejected && note == "" {
			problems = append(problems, "say what was fixed")
		}
	}
	return problems
}

// Apply moves the process. The caller has already checked CanMove and
// CheckStep.
func (a *Acceptance) Apply(party DisputeParty, actorID string, s AcceptanceStep, now time.Time) {
	from := a.Stage
	note := strings.TrimSpace(s.Note)
	a.UpdatedAt = now
	if s.NextAction != "" {
		a.NextAction = strings.TrimSpace(s.NextAction)
	}
	if s.DueAt != nil {
		a.DueAt = s.DueAt
	}

	if s.To == StageOnHold {
		a.HeldFrom, a.Stage, a.HoldReason = from, StageOnHold, note
		return
	}
	if from == StageOnHold {
		// Resuming: back to the stage it left, with the hold cleared.
		a.Stage, a.HeldFrom, a.HoldReason = s.To, "", ""
		return
	}
	a.Stage = s.To
	switch s.To {
	case StageBuilt:
		a.BuiltAt, a.Reason, a.Responsible = &now, "", PartyColoplus
	case StageReadyForInspection:
		a.ReadyAt, a.Dossier, a.Reason, a.Responsible = &now, note, "", PartyAirtel
	case StageInspected:
		a.InspectedAt = &now
	case StageRejected:
		a.Reason, a.Responsible = note, PartyColoplus
	case StageAccepted:
		a.DecidedAt, a.DecidedBy = &now, actorID
	}
}

// AcceptanceEvent is one entry in the process's history.
type AcceptanceEvent struct {
	ID           string
	AcceptanceID string
	AuthorID     string
	Party        DisputeParty
	FromStage    AcceptanceStage
	ToStage      AcceptanceStage
	Body         string
	CreatedAt    time.Time
}

// AcceptanceParty decides which side the actor speaks for, by permission.
func AcceptanceParty(a Actor) (DisputeParty, bool) {
	switch {
	case a.Can(PermAcceptanceManage):
		return PartyColoplus, true
	case a.Can(PermAcceptanceReview):
		return PartyAirtel, true
	}
	return "", false
}
