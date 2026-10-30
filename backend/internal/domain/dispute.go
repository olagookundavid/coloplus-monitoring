package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type DisputeStatus string

const (
	DisputeOpen             DisputeStatus = "open"
	DisputeUnderReview      DisputeStatus = "under_review"
	DisputeAwaitingEvidence DisputeStatus = "awaiting_evidence"
	DisputeAgreedCorrection DisputeStatus = "agreed_correction"
	DisputeResolved         DisputeStatus = "resolved"
	DisputeClosed           DisputeStatus = "closed"
)

func ValidDisputeStatus(s DisputeStatus) bool {
	switch s {
	case DisputeOpen, DisputeUnderReview, DisputeAwaitingEvidence, DisputeAgreedCorrection,
		DisputeResolved, DisputeClosed:
		return true
	}
	return false
}

type DisputeCategory string

const (
	DisputeIncorrectOutage DisputeCategory = "incorrect_outage"
	DisputeMissingOutage   DisputeCategory = "missing_outage"
	DisputeIncorrectServed DisputeCategory = "incorrect_served"
	DisputeWrongDuration   DisputeCategory = "wrong_duration"
	DisputeWrongMetric     DisputeCategory = "wrong_metric"
	DisputeMissingReport   DisputeCategory = "missing_report"
	DisputeOther           DisputeCategory = "other"
)

func ValidDisputeCategory(c DisputeCategory) bool {
	switch c {
	case DisputeIncorrectOutage, DisputeMissingOutage, DisputeIncorrectServed, DisputeWrongDuration,
		DisputeWrongMetric, DisputeMissingReport, DisputeOther:
		return true
	}
	return false
}

// DisputeMetric names the figure being disputed, when one is.
type DisputeMetric string

const (
	MetricNone          DisputeMetric = ""
	MetricAvailability  DisputeMetric = "availability"
	MetricCoverage      DisputeMetric = "coverage"
	MetricOutageMinutes DisputeMetric = "outage_minutes"
	MetricServedMinutes DisputeMetric = "served_minutes"
	MetricOther         DisputeMetric = "other"
)

func ValidDisputeMetric(m DisputeMetric) bool {
	switch m {
	case MetricNone, MetricAvailability, MetricCoverage, MetricOutageMinutes, MetricServedMinutes, MetricOther:
		return true
	}
	return false
}

// DisputeOutcome is how Coloplus settled it, or withdrawn when Airtel dropped it.
type DisputeOutcome string

const (
	OutcomeNone      DisputeOutcome = ""
	OutcomeUpheld    DisputeOutcome = "upheld"
	OutcomeNotUpheld DisputeOutcome = "not_upheld"
	OutcomeWithdrawn DisputeOutcome = "withdrawn"
)

// DisputeParty is which side of the dispute a person is on. Their permissions
// decide it, never their name: Coloplus staff manage, Airtel reviewers raise.
type DisputeParty string

const (
	PartyAirtel   DisputeParty = "airtel"
	PartyColoplus DisputeParty = "coloplus"
)

// PartyOf says which side the actor speaks for; false when neither.
func PartyOf(a Actor) (DisputeParty, bool) {
	switch {
	case a.Can(PermDisputesManage):
		return PartyColoplus, true
	case a.Can(PermDisputesRaise):
		return PartyAirtel, true
	}
	return "", false
}

const (
	maxDisputeTitle  = 200
	maxDisputeText   = 4000
	maxDisputePeriod = 366 * 24 * time.Hour
)

type Dispute struct {
	ID       string
	Number   int64
	SiteID   string
	ReportID string
	// IntervalID is set only together with ReportID.
	IntervalID  string
	Metric      DisputeMetric
	PeriodFrom  *time.Time
	PeriodTo    *time.Time
	Category    DisputeCategory
	Title       string
	Description string
	Status      DisputeStatus
	RaisedBy    string
	OwnerID     string

	Outcome            DisputeOutcome
	Resolution         string
	ResolutionReportID string
	ResolvedAt         *time.Time
	ResolvedBy         string

	AcknowledgedAt *time.Time
	AcknowledgedBy string

	ReopenCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Open reports whether the dispute is still being handled.
func (d Dispute) Open() bool { return d.Status != DisputeClosed }

// WaitingOn says whose move it is: Coloplus while it reviews or corrects,
// Airtel while it owes evidence or an acknowledgement. Nobody once closed.
func (d Dispute) WaitingOn() DisputeParty {
	switch d.Status {
	case DisputeOpen, DisputeUnderReview, DisputeAgreedCorrection:
		return PartyColoplus
	case DisputeAwaitingEvidence, DisputeResolved:
		return PartyAirtel
	}
	return ""
}

// ValidateNew lists what is wrong with a dispute about to be raised.
func (d Dispute) ValidateNew(today time.Time) []string {
	var problems []string
	if !ValidDisputeCategory(d.Category) {
		problems = append(problems, "choose what is wrong: incorrect outage, missing outage, incorrect served time, wrong duration, wrong metric, missing report or other")
	}
	if !ValidDisputeMetric(d.Metric) {
		problems = append(problems, "choose a valid metric")
	}
	if strings.TrimSpace(d.Title) == "" {
		problems = append(problems, "give the dispute a title")
	} else if utf8.RuneCountInString(d.Title) > maxDisputeTitle {
		problems = append(problems, "the title is too long")
	}
	if utf8.RuneCountInString(d.Description) > maxDisputeText {
		problems = append(problems, "the description is too long")
	}
	if d.IntervalID != "" && d.ReportID == "" {
		problems = append(problems, "an interval can only be disputed through its report")
	}
	if (d.PeriodFrom == nil) != (d.PeriodTo == nil) {
		problems = append(problems, "give both ends of the period, or neither")
	} else if d.PeriodFrom != nil {
		switch {
		case d.PeriodTo.Before(*d.PeriodFrom):
			problems = append(problems, "the period ends before it starts")
		case d.PeriodTo.Sub(*d.PeriodFrom) > maxDisputePeriod:
			problems = append(problems, "a dispute covers at most a year; raise a second one for the rest")
		case d.PeriodTo.After(today):
			problems = append(problems, "the period cannot reach into the future")
		}
	}
	return problems
}

// DisputeStep is one move along the workflow.
type DisputeStep struct {
	To   DisputeStatus
	Note string
	// Outcome is required to resolve: upheld or not_upheld.
	Outcome DisputeOutcome
	// ResolutionReportID is the corrected report revision that settles an
	// upheld dispute about a report.
	ResolutionReportID string
}

// CanMove says whether the party may take the dispute from where it is to the
// step's status. Coloplus reviews and settles; Airtel withdraws, supplies
// evidence, reopens or acknowledges. Nobody may leave closed.
func (d Dispute) CanMove(party DisputeParty, to DisputeStatus) bool {
	switch party {
	case PartyColoplus:
		switch d.Status {
		case DisputeOpen:
			return to == DisputeUnderReview
		case DisputeUnderReview:
			return to == DisputeAwaitingEvidence || to == DisputeAgreedCorrection || to == DisputeResolved
		case DisputeAwaitingEvidence:
			return to == DisputeUnderReview || to == DisputeResolved
		case DisputeAgreedCorrection:
			return to == DisputeResolved
		}
	case PartyAirtel:
		switch d.Status {
		case DisputeOpen, DisputeUnderReview:
			return to == DisputeClosed
		case DisputeAwaitingEvidence:
			return to == DisputeClosed || to == DisputeUnderReview
		case DisputeResolved:
			return to == DisputeClosed || to == DisputeUnderReview
		}
	}
	return false
}

// CheckStep lists what is missing or wrong in the step, apart from whether
// the move is allowed at all.
func (d Dispute) CheckStep(party DisputeParty, s DisputeStep) []string {
	var problems []string
	note := strings.TrimSpace(s.Note)
	if utf8.RuneCountInString(note) > maxDisputeText {
		problems = append(problems, "the note is too long")
	}
	switch {
	case party == PartyColoplus && s.To == DisputeAwaitingEvidence:
		if note == "" {
			problems = append(problems, "say what evidence is needed")
		}
	case party == PartyColoplus && s.To == DisputeAgreedCorrection:
		if note == "" {
			problems = append(problems, "say what will be corrected")
		}
	case party == PartyColoplus && s.To == DisputeResolved:
		if note == "" {
			problems = append(problems, "explain the outcome")
		}
		switch s.Outcome {
		case OutcomeUpheld:
			if d.Status != DisputeAgreedCorrection {
				problems = append(problems, "agree the correction first, then resolve once it is made")
			}
			if d.ReportID != "" && s.ResolutionReportID == "" {
				problems = append(problems, "name the corrected report that settles it")
			}
		case OutcomeNotUpheld:
			if d.Status == DisputeAgreedCorrection {
				problems = append(problems, "a dispute whose correction was agreed has to end upheld")
			}
		default:
			problems = append(problems, "say whether the dispute is upheld or not upheld")
		}
	case party == PartyAirtel && s.To == DisputeUnderReview:
		if note == "" {
			problems = append(problems, "say what changed: the new evidence, or why you do not accept the resolution")
		}
	case party == PartyAirtel && s.To == DisputeClosed && d.Status != DisputeResolved:
		if note == "" {
			problems = append(problems, "say why you are withdrawing it")
		}
	}
	return problems
}

// Apply moves the dispute. The caller has already checked CanMove and
// CheckStep. A dispute taken for review by a person is theirs from then on.
func (d *Dispute) Apply(party DisputeParty, actorID string, s DisputeStep, now time.Time) {
	from := d.Status
	d.Status = s.To
	d.UpdatedAt = now
	switch {
	case party == PartyColoplus && s.To == DisputeUnderReview && d.OwnerID == "":
		d.OwnerID = actorID
	case party == PartyColoplus && s.To == DisputeResolved:
		d.Outcome, d.Resolution = s.Outcome, strings.TrimSpace(s.Note)
		if s.Outcome == OutcomeUpheld {
			d.ResolutionReportID = s.ResolutionReportID
		}
		d.ResolvedAt, d.ResolvedBy = &now, actorID
	case party == PartyAirtel && s.To == DisputeUnderReview && from == DisputeResolved:
		// Reopened: the resolution no longer stands.
		d.Outcome, d.Resolution, d.ResolutionReportID = OutcomeNone, "", ""
		d.ResolvedAt, d.ResolvedBy = nil, ""
		d.ReopenCount++
	case party == PartyAirtel && s.To == DisputeClosed && from == DisputeResolved:
		d.AcknowledgedAt, d.AcknowledgedBy = &now, actorID
	case party == PartyAirtel && s.To == DisputeClosed:
		d.Outcome = OutcomeWithdrawn
	}
}

// DisputeEvent is one entry in the dispute's history.
type DisputeEvent struct {
	ID         string
	DisputeID  string
	AuthorID   string
	Party      DisputeParty
	Kind       DisputeEventKind
	FromStatus DisputeStatus
	ToStatus   DisputeStatus
	Body       string
	// Internal entries are Coloplus's own notes and are never shown to Airtel.
	Internal  bool
	CreatedAt time.Time
}

type DisputeEventKind string

const (
	EventComment DisputeEventKind = "comment"
	EventStatus  DisputeEventKind = "status"
)

// ValidateComment checks a comment before it is stored.
func ValidateComment(body string) []string {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return []string{"write the comment"}
	case utf8.RuneCountInString(body) > maxDisputeText:
		return []string{"the comment is too long"}
	}
	return nil
}
