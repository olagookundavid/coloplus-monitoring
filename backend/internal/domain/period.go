package domain

import "time"

// PeriodClosure is a month that was closed, and perhaps reopened since.
type PeriodClosure struct {
	ID string
	// Month is the first day of the month, as a calendar date.
	Month    time.Time
	ClosedBy string
	ClosedAt time.Time
	// RunID is the calculation run stored when the month was closed.
	RunID        string
	ReopenedBy   string
	ReopenedAt   *time.Time
	ReopenReason string
}

// Open reports whether the closure still locks the month.
func (c PeriodClosure) Open() bool { return c.ReopenedAt == nil }

// MonthOf returns the first day of the month a date falls in.
func MonthOf(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
}
