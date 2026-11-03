package dto

import "time"

// PeriodClosureDTO is a month that was closed. While reopened_at is empty the
// month is locked: no report for any of its days can be added, corrected or approved.
type PeriodClosureDTO struct {
	ID       string    `json:"id"`
	Month    string    `json:"month" example:"2026-10"`
	ClosedBy string    `json:"closed_by"`
	ClosedAt time.Time `json:"closed_at"`
	// RunID is the calculation run stored when the month was closed.
	RunID        string     `json:"run_id"`
	Open         bool       `json:"locked"`
	ReopenedBy   string     `json:"reopened_by,omitempty"`
	ReopenedAt   *time.Time `json:"reopened_at,omitempty"`
	ReopenReason string     `json:"reopen_reason,omitempty"`
}

type PeriodListResponse struct {
	Periods []PeriodClosureDTO `json:"periods"`
}

// ReopenPeriodRequest says why a closed month is being reopened.
type ReopenPeriodRequest struct {
	Reason string `json:"reason"`
}
