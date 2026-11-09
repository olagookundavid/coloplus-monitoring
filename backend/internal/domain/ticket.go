package domain

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type TicketStatus string

const (
	TicketOpen       TicketStatus = "open"
	TicketInProgress TicketStatus = "in_progress"
	TicketResolved   TicketStatus = "resolved"
)

func ValidTicketStatus(s TicketStatus) bool {
	switch s {
	case TicketOpen, TicketInProgress, TicketResolved:
		return true
	}
	return false
}

type TicketCategory string

const (
	TicketPowerOutage       TicketCategory = "power_outage"
	TicketLowFuel           TicketCategory = "low_fuel"
	TicketGeneratorFault    TicketCategory = "generator_fault"
	TicketEquipmentFault    TicketCategory = "equipment_fault"
	TicketAccessRestriction TicketCategory = "access_restriction"
	TicketOther             TicketCategory = "other"
)

func ValidTicketCategory(c TicketCategory) bool {
	switch c {
	case TicketPowerOutage, TicketLowFuel, TicketGeneratorFault, TicketEquipmentFault, TicketAccessRestriction, TicketOther:
		return true
	}
	return false
}

// TicketPriority is what the requester asked for. It is not an incident
// severity and Coloplus does not read it as one.
type TicketPriority string

const (
	TicketLow      TicketPriority = "low"
	TicketMedium   TicketPriority = "medium"
	TicketHigh     TicketPriority = "high"
	TicketCritical TicketPriority = "critical"
)

func ValidTicketPriority(p TicketPriority) bool {
	switch p {
	case TicketLow, TicketMedium, TicketHigh, TicketCritical:
		return true
	}
	return false
}

// TicketPartyOf says which side the actor speaks for: Coloplus works tickets,
// The customer raises them. The party values are shared with disputes.
func TicketPartyOf(a Actor) (DisputeParty, bool) {
	switch {
	case a.Can(PermTicketsManage):
		return PartyColoplus, true
	case a.Can(PermTicketsRaise):
		return PartyCustomer, true
	}
	return "", false
}

const (
	maxTicketSubject = 200
	maxTicketText    = 4000
	maxEmailLen      = 254
)

type Ticket struct {
	ID           string
	Number       int64
	SiteID       string
	Category     TicketCategory
	Priority     TicketPriority
	Subject      string
	Description  string
	ContactEmail string
	Status       TicketStatus
	RaisedBy     string
	AssigneeID   string

	Resolution  string
	ResolvedAt  *time.Time
	ResolvedBy  string
	ReopenCount int

	// ClientKey makes a retried request land on the ticket it already made.
	ClientKey string

	// IncidentID is the incident this ticket became or belongs to. The other
	// two fields are read with it so that the customer is only told about an incident
	// that has been published to them.
	IncidentID        string
	IncidentNumber    int64
	IncidentPublished bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Open reports whether the ticket still needs work.
func (t Ticket) Open() bool { return t.Status != TicketResolved }

// ValidateNew lists what is wrong with a ticket about to be raised.
func (t Ticket) ValidateNew() []string {
	var problems []string
	if !ValidTicketCategory(t.Category) {
		problems = append(problems, "choose a category: power outage, low fuel, generator fault, equipment fault, access restriction or other")
	}
	if !ValidTicketPriority(t.Priority) {
		problems = append(problems, "choose a priority: low, medium, high or critical")
	}
	switch {
	case strings.TrimSpace(t.Subject) == "":
		problems = append(problems, "give the ticket a subject")
	case utf8.RuneCountInString(t.Subject) > maxTicketSubject:
		problems = append(problems, "the subject is too long")
	}
	switch {
	case strings.TrimSpace(t.Description) == "":
		problems = append(problems, "say what happened")
	case utf8.RuneCountInString(t.Description) > maxTicketText:
		problems = append(problems, "the description is too long")
	}
	if t.ContactEmail != "" {
		if a, err := mail.ParseAddress(t.ContactEmail); err != nil || a.Address != t.ContactEmail || len(t.ContactEmail) > maxEmailLen {
			problems = append(problems, "the contact email does not look right")
		}
	}
	return problems
}

// TicketStep is one move along the workflow.
type TicketStep struct {
	To   TicketStatus
	Note string
}

// CanMove says whether the party may take the ticket to the step's status.
// Coloplus starts and resolves, and can take a resolved ticket back into work;
// The customer can only reopen a resolved one.
func (t Ticket) CanMove(party DisputeParty, to TicketStatus) bool {
	switch party {
	case PartyColoplus:
		switch t.Status {
		case TicketOpen:
			return to == TicketInProgress || to == TicketResolved
		case TicketInProgress:
			return to == TicketResolved
		case TicketResolved:
			return to == TicketInProgress
		}
	case PartyCustomer:
		return t.Status == TicketResolved && to == TicketOpen
	}
	return false
}

// CheckStep lists what is missing in the step, apart from whether the move is
// allowed at all.
func (t Ticket) CheckStep(party DisputeParty, s TicketStep) []string {
	note := strings.TrimSpace(s.Note)
	if utf8.RuneCountInString(note) > maxTicketText {
		return []string{"the note is too long"}
	}
	switch {
	case party == PartyColoplus && s.To == TicketResolved && note == "":
		return []string{"say how it was resolved"}
	case party == PartyCustomer && s.To == TicketOpen && note == "":
		return []string{"say why it is not resolved"}
	case party == PartyColoplus && t.Status == TicketResolved && s.To == TicketInProgress && note == "":
		return []string{"say why the ticket is being worked again"}
	}
	return nil
}

// Apply moves the ticket. The caller has already checked CanMove and
// CheckStep. Whoever starts a ticket owns it unless someone already does.
func (t *Ticket) Apply(party DisputeParty, actorID string, s TicketStep, now time.Time) {
	from := t.Status
	t.Status = s.To
	t.UpdatedAt = now
	switch {
	case party == PartyColoplus && s.To == TicketInProgress && from == TicketResolved:
		t.Resolution, t.ResolvedAt, t.ResolvedBy = "", nil, ""
		t.ReopenCount++
	case party == PartyColoplus && s.To == TicketInProgress && t.AssigneeID == "":
		t.AssigneeID = actorID
	case party == PartyColoplus && s.To == TicketResolved:
		t.Resolution, t.ResolvedAt, t.ResolvedBy = strings.TrimSpace(s.Note), &now, actorID
		if t.AssigneeID == "" {
			t.AssigneeID = actorID
		}
	case party == PartyCustomer && s.To == TicketOpen:
		t.Resolution, t.ResolvedAt, t.ResolvedBy = "", nil, ""
		t.ReopenCount++
	}
}

type TicketEventKind string

const (
	TicketEventComment TicketEventKind = "comment"
	TicketEventStatus  TicketEventKind = "status"
	TicketEventLink    TicketEventKind = "link"
)

// TicketEvent is one entry in the ticket's history.
type TicketEvent struct {
	ID         string
	TicketID   string
	AuthorID   string
	Party      DisputeParty
	Kind       TicketEventKind
	FromStatus TicketStatus
	ToStatus   TicketStatus
	Body       string
	// Internal entries are Coloplus's own notes and never reach the customer.
	Internal  bool
	CreatedAt time.Time
}

// ValidateTicketComment checks a comment before it is stored.
func ValidateTicketComment(body string) []string {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return []string{"write the comment"}
	case utf8.RuneCountInString(body) > maxTicketText:
		return []string{"the comment is too long"}
	}
	return nil
}
