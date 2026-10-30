package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type IncidentStatus string

const (
	IncidentReported     IncidentStatus = "reported"
	IncidentAcknowledged IncidentStatus = "acknowledged"
	IncidentAssigned     IncidentStatus = "assigned"
	IncidentInProgress   IncidentStatus = "in_progress"
	IncidentRestored     IncidentStatus = "restored"
	IncidentVerified     IncidentStatus = "verified"
	IncidentClosed       IncidentStatus = "closed"
)

type IncidentCategory string

const (
	CategoryGridLoss           IncidentCategory = "grid_loss"
	CategoryBuildingSupplyLoss IncidentCategory = "building_supply_loss"
	CategoryGeneratorFailure   IncidentCategory = "generator_failure"
	CategoryLowFuel            IncidentCategory = "low_fuel"
	CategoryBatteryFailure     IncidentCategory = "battery_failure"
	CategoryRectifierFault     IncidentCategory = "rectifier_fault"
	CategoryElectricalFault    IncidentCategory = "electrical_fault"
	CategoryAccessRestriction  IncidentCategory = "access_restriction"
	CategoryOther              IncidentCategory = "other"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityMajor    Severity = "major"
	SeverityMinor    Severity = "minor"
)

// Impact separates a service interruption from a fault that caused none.
type Impact string

const (
	ImpactOutage   Impact = "outage"
	ImpactNoOutage Impact = "no_outage"
)

func ValidIncidentCategory(c IncidentCategory) bool {
	switch c {
	case CategoryGridLoss, CategoryBuildingSupplyLoss, CategoryGeneratorFailure, CategoryLowFuel,
		CategoryBatteryFailure, CategoryRectifierFault, CategoryElectricalFault,
		CategoryAccessRestriction, CategoryOther:
		return true
	}
	return false
}

func ValidSeverity(s Severity) bool {
	return s == SeverityCritical || s == SeverityMajor || s == SeverityMinor
}

func ValidImpact(i Impact) bool { return i == ImpactOutage || i == ImpactNoOutage }

func ValidIncidentStatus(s IncidentStatus) bool {
	switch s {
	case IncidentReported, IncidentAcknowledged, IncidentAssigned, IncidentInProgress,
		IncidentRestored, IncidentVerified, IncidentClosed:
		return true
	}
	return false
}

const (
	maxIncidentTitle = 120
	maxIncidentText  = 4000
)

// Incident is one reported fault.
type Incident struct {
	ID       string
	Number   int64
	SiteID   string
	Category IncidentCategory
	Severity Severity
	Impact   Impact
	Title    string
	// Description is internal; Airtel sees only the customer updates.
	Description string
	Status      IncidentStatus

	StartedAt  time.Time
	DetectedAt time.Time
	ReporterID string
	OwnerID    string

	NextAction    string
	NextUpdateDue *time.Time
	// RestorationTarget is a plan. It never counts as restoration.
	RestorationTarget *time.Time

	RestoredAt *time.Time
	RestoredBy string
	VerifiedAt *time.Time
	VerifiedBy string
	ClosedAt   *time.Time
	RootCause  string

	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IncidentUpdate is one line of an incident's history.
type IncidentUpdate struct {
	ID              string
	IncidentID      string
	AuthorID        string
	Kind            UpdateKind
	FromStatus      IncidentStatus
	ToStatus        IncidentStatus
	Body            string
	CustomerVisible bool
	ObservedAt      *time.Time
	CreatedAt       time.Time
}

type UpdateKind string

const (
	UpdateNote       UpdateKind = "note"
	UpdateStatus     UpdateKind = "status"
	UpdateAssignment UpdateKind = "assignment"
	UpdatePlan       UpdateKind = "plan"
	UpdateCustomer   UpdateKind = "customer"
)

// ValidateNew checks a freshly reported incident and returns every problem.
func (i Incident) ValidateNew(now time.Time) []string {
	var p []string
	title := strings.TrimSpace(i.Title)
	switch {
	case title == "":
		p = append(p, "give the incident a short title")
	case utf8.RuneCountInString(title) > maxIncidentTitle:
		p = append(p, fmt.Sprintf("the title can be at most %d characters", maxIncidentTitle))
	}
	if utf8.RuneCountInString(i.Description) > maxIncidentText {
		p = append(p, fmt.Sprintf("the description can be at most %d characters", maxIncidentText))
	}
	if !ValidIncidentCategory(i.Category) {
		p = append(p, "choose a valid category")
	}
	if !ValidSeverity(i.Severity) {
		p = append(p, "severity must be critical, major or minor")
	}
	if !ValidImpact(i.Impact) {
		p = append(p, "impact must be outage or no_outage")
	}
	switch {
	case i.StartedAt.IsZero():
		p = append(p, "say when the fault started")
	case i.StartedAt.After(now):
		p = append(p, "the fault cannot have started in the future")
	}
	if !i.StartedAt.IsZero() {
		if i.DetectedAt.Before(i.StartedAt) {
			p = append(p, "the fault cannot be detected before it started")
		}
		if i.DetectedAt.After(now) {
			p = append(p, "the detection time cannot be in the future")
		}
	}
	return p
}

// Open means not yet closed. Active means service is still affected or
// awaiting restoration.
func (i Incident) Open() bool   { return i.Status != IncidentClosed }
func (i Incident) Active() bool { return i.Open() && !i.Restored() }

// Restored is true once restoration has been recorded, whatever comes after.
func (i Incident) Restored() bool {
	switch i.Status {
	case IncidentRestored, IncidentVerified, IncidentClosed:
		return true
	}
	return false
}

// Overdue is true when the promised next update has not come.
func (i Incident) Overdue(now time.Time) bool {
	return i.Open() && i.NextUpdateDue != nil && now.After(*i.NextUpdateDue)
}

// AssignmentAdvances reports whether naming an owner also moves the incident
// to the assigned stage.
func (i Incident) AssignmentAdvances() bool {
	return i.Status == IncidentReported || i.Status == IncidentAcknowledged
}

var incidentTransitions = map[IncidentStatus][]IncidentStatus{
	IncidentReported:     {IncidentAcknowledged, IncidentInProgress, IncidentRestored},
	IncidentAcknowledged: {IncidentInProgress, IncidentRestored},
	IncidentAssigned:     {IncidentInProgress, IncidentRestored},
	IncidentInProgress:   {IncidentRestored},
	IncidentRestored:     {IncidentVerified, IncidentInProgress},
	IncidentVerified:     {IncidentClosed, IncidentInProgress},
}

// CanMoveTo reports whether the workflow allows the step. Assigned is reached
// by naming an owner, not by a status change.
func (i Incident) CanMoveTo(to IncidentStatus) bool {
	for _, next := range incidentTransitions[i.Status] {
		if next == to {
			return true
		}
	}
	return false
}

// Transition describes a requested status change.
type Transition struct {
	To IncidentStatus
	// At is when restoration happened, or when the verifying observation was
	// made. Required for restored and verified.
	At        *time.Time
	Note      string
	RootCause string
}

// CheckTransition returns what is wrong with the request. It assumes the
// step itself is allowed (CanMoveTo).
func (i Incident) CheckTransition(t Transition, actorID string, now time.Time) []string {
	var p []string
	if utf8.RuneCountInString(t.Note) > maxIncidentText {
		p = append(p, fmt.Sprintf("the note can be at most %d characters", maxIncidentText))
	}
	switch t.To {
	case IncidentInProgress:
		if i.OwnerID == "" {
			p = append(p, "assign an owner before work starts")
		}
		if i.Restored() && strings.TrimSpace(t.Note) == "" {
			p = append(p, "say why the incident is being reopened")
		}
	case IncidentRestored:
		switch {
		case t.At == nil:
			p = append(p, "say when power was actually restored; a target time does not count")
		case t.At.Before(i.StartedAt):
			p = append(p, "restoration cannot be before the fault started")
		case t.At.After(now):
			p = append(p, "restoration cannot be in the future")
		}
	case IncidentVerified:
		if strings.TrimSpace(t.Note) == "" {
			p = append(p, "describe the new observation or evidence that confirms restoration")
		}
		switch {
		case t.At == nil:
			p = append(p, "say when the confirming observation was made")
		case i.RestoredAt != nil && t.At.Before(*i.RestoredAt):
			p = append(p, "the confirming observation must be after restoration was recorded")
		case t.At.After(now):
			p = append(p, "the observation cannot be in the future")
		}
		if i.RestoredBy == actorID {
			p = append(p, "restoration must be verified by someone other than the person who recorded it")
		}
	case IncidentClosed:
		if strings.TrimSpace(t.RootCause) == "" {
			p = append(p, "record the root cause before closing")
		}
	}
	return p
}

// ApplyTransition moves the incident. Call it only after CanMoveTo and
// CheckTransition have passed.
func (i *Incident) ApplyTransition(t Transition, actorID string, now time.Time) {
	from := i.Status
	i.Status = t.To
	i.UpdatedAt = now
	switch t.To {
	case IncidentRestored:
		at := t.At.UTC()
		i.RestoredAt, i.RestoredBy = &at, actorID
	case IncidentVerified:
		// The time of the confirming observation, not of the data entry.
		at := t.At.UTC()
		i.VerifiedAt, i.VerifiedBy = &at, actorID
	case IncidentClosed:
		at := now
		i.ClosedAt = &at
		i.RootCause = strings.TrimSpace(t.RootCause)
	case IncidentInProgress:
		// Reopening discards the earlier restoration and verification; the
		// history keeps what was recorded.
		if from == IncidentRestored || from == IncidentVerified {
			i.RestoredAt, i.RestoredBy = nil, ""
			i.VerifiedAt, i.VerifiedBy = nil, ""
		}
	}
}
