package domain

import (
	"fmt"
	"strings"
	"time"
)

type MaintenanceStatus string

const (
	MaintenanceRequested MaintenanceStatus = "requested"
	MaintenanceApproved  MaintenanceStatus = "approved"
	MaintenanceRejected  MaintenanceStatus = "rejected"
	MaintenanceCancelled MaintenanceStatus = "cancelled"
)

func ValidMaintenanceStatus(s MaintenanceStatus) bool {
	switch s {
	case MaintenanceRequested, MaintenanceApproved, MaintenanceRejected, MaintenanceCancelled:
		return true
	}
	return false
}

// MaintenanceWindow is a period of planned work at a site. Once approved it is
// left out of the site's eligible time.
type MaintenanceWindow struct {
	ID          string
	SiteID      string
	StartsAt    time.Time
	EndsAt      time.Time
	Reason      string
	Status      MaintenanceStatus
	RequestedBy string
	RequestedAt time.Time
	DecidedBy   string
	DecidedAt   *time.Time
	// DecisionNote is why it was approved, rejected or cancelled.
	DecisionNote string
}

func (w MaintenanceWindow) Span() Span { return Span{w.StartsAt, w.EndsAt} }

// Duration of the window.
func (w MaintenanceWindow) Duration() time.Duration { return w.EndsAt.Sub(w.StartsAt) }

// MaintenanceRules are the limits a request must meet. The numbers are
// proposals until the customer agrees them.
type MaintenanceRules struct {
	// NoticeHours is how far ahead of the start a window must be asked for.
	NoticeHours int
	// MaxHours is the longest single window.
	MaxHours int
	// MonthlyCapHours is the most planned maintenance a site may have in a month.
	MonthlyCapHours int
}

// Validate checks a request against the rules, in the portal's time zone.
// usedThisMonth is the time already requested or approved for the site in the
// month the window starts in.
func (r MaintenanceRules) Validate(start, end time.Time, reason string, now time.Time, loc *time.Location, usedThisMonth time.Duration) []string {
	var problems []string
	if len(strings.TrimSpace(reason)) < 5 {
		problems = append(problems, "say what the maintenance is")
	}
	if !end.After(start) {
		problems = append(problems, "the end must be after the start")
		return problems
	}
	if start.Before(now.Add(time.Duration(r.NoticeHours) * time.Hour)) {
		problems = append(problems, fmt.Sprintf("planned maintenance must be asked for at least %d hours ahead", r.NoticeHours))
	}
	if end.Sub(start) > time.Duration(r.MaxHours)*time.Hour {
		problems = append(problems, fmt.Sprintf("a window can be at most %d hours", r.MaxHours))
	}
	sy, sm, sd := start.In(loc).Date()
	ey, em, ed := end.Add(-time.Nanosecond).In(loc).Date()
	if sy != ey || sm != em || sd != ed {
		problems = append(problems, "a window must stay within one day; ask for each day separately")
	}
	if usedThisMonth+end.Sub(start) > time.Duration(r.MonthlyCapHours)*time.Hour {
		problems = append(problems, fmt.Sprintf("that would take the site over %d hours of planned maintenance this month", r.MonthlyCapHours))
	}
	return problems
}
