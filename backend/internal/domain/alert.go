package domain

import "time"

// Alert is something a person should know about, kept until they acknowledge it.
type Alert struct {
	ID             string
	UserID         string
	Kind           string
	Title          string
	Body           string
	CreatedAt      time.Time
	AcknowledgedAt *time.Time
}
