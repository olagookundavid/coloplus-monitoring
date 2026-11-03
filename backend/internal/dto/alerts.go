package dto

import "time"

type AlertDTO struct {
	ID    string `json:"id"`
	Kind  string `json:"kind" example:"fuel.low"`
	Title string `json:"title"`
	// Body is the full text of the alert, the same as the email.
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	// AcknowledgedAt is null until the person acknowledges it.
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
}

type AlertListResponse struct {
	Alerts []AlertDTO `json:"alerts"`
	// Total counts the alerts matching the request; Unread all open alerts.
	Total  int `json:"total"`
	Unread int `json:"unread"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type AcknowledgeAllResponse struct {
	Acknowledged int `json:"acknowledged"`
}

// DigestPreferenceDTO is whether the person gets the weekly summary email.
type DigestPreferenceDTO struct {
	Weekly bool `json:"weekly"`
}
