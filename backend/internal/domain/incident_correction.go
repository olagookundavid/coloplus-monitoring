package domain

import (
	"strings"
	"time"
)

// CorrectionField names the one incident field a correction can target.
type CorrectionField string

const (
	FieldStartedAt  CorrectionField = "started_at"
	FieldRestoredAt CorrectionField = "restored_at"
)

func ValidCorrectionField(f CorrectionField) bool {
	return f == FieldStartedAt || f == FieldRestoredAt
}

type CorrectionStatus string

const (
	CorrectionRequested CorrectionStatus = "requested"
	CorrectionApproved  CorrectionStatus = "approved"
	CorrectionRejected  CorrectionStatus = "rejected"
)

// IncidentCorrection is a reporter's request to change a single already-
// recorded incident field. The original value is kept permanently, even
// once approved.
type IncidentCorrection struct {
	ID             string
	Number         int64
	IncidentID     string
	SiteID         string
	Field          CorrectionField
	OriginalValue  time.Time
	CorrectedValue time.Time
	Reason         string
	Status         CorrectionStatus
	RequestedBy    string
	ReviewedBy     string
	ReviewedAt     *time.Time
	ReviewNote     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Open reports whether the correction is still waiting on a decision.
func (c IncidentCorrection) Open() bool { return c.Status == CorrectionRequested }

// ValidateNew checks a correction before it is stored. original is the
// field's current value on the live incident, read inside the caller's
// transaction -- never supplied by the client.
func (c IncidentCorrection) ValidateNew(now time.Time) []string {
	var p []string
	if !ValidCorrectionField(c.Field) {
		p = append(p, "choose a field that can be corrected")
	}
	if strings.TrimSpace(c.Reason) == "" {
		p = append(p, "explain why this needs correcting")
	}
	if c.CorrectedValue.Equal(c.OriginalValue) {
		p = append(p, "the corrected value must actually differ from the current one")
	}
	if c.CorrectedValue.After(now) {
		p = append(p, "the corrected time cannot be in the future")
	}
	return p
}

// CheckApply re-validates the correction against the incident's live state
// at approval time (it may have moved since the request was filed): the
// same ordering rules Incident.CheckTransition applies, replayed for
// whichever single field is being corrected.
func (c IncidentCorrection) CheckApply(i Incident) []string {
	var p []string
	switch c.Field {
	case FieldStartedAt:
		if c.CorrectedValue.After(i.DetectedAt) {
			p = append(p, "the fault cannot be detected before it started")
		}
		if i.RestoredAt != nil && c.CorrectedValue.After(*i.RestoredAt) {
			p = append(p, "the fault cannot start after it was restored")
		}
	case FieldRestoredAt:
		if i.RestoredAt == nil {
			p = append(p, "this incident has no restoration time to correct")
		}
		if c.CorrectedValue.Before(i.StartedAt) {
			p = append(p, "restoration cannot be before the fault started")
		}
		if i.VerifiedAt != nil && c.CorrectedValue.After(*i.VerifiedAt) {
			p = append(p, "restoration cannot be after it was verified")
		}
	}
	return p
}

// ApplyTo sets the corrected value on the incident. Call only after
// CheckApply has passed.
func (c IncidentCorrection) ApplyTo(i *Incident) {
	switch c.Field {
	case FieldStartedAt:
		i.StartedAt = c.CorrectedValue
	case FieldRestoredAt:
		v := c.CorrectedValue
		i.RestoredAt = &v
	}
}

// FieldValue reads a field's current value off an incident, for the
// snapshot taken when a correction is requested. ok is false for
// restored_at when the incident has not been restored yet.
func FieldValue(i Incident, f CorrectionField) (time.Time, bool) {
	switch f {
	case FieldStartedAt:
		return i.StartedAt, true
	case FieldRestoredAt:
		if i.RestoredAt == nil {
			return time.Time{}, false
		}
		return *i.RestoredAt, true
	}
	return time.Time{}, false
}
