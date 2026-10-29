package dto

import "time"

type AuditEventDTO struct {
	ID         int64          `json:"id"`
	OccurredAt time.Time      `json:"occurred_at"`
	ActorID    string         `json:"actor_id,omitempty"`
	Action     string         `json:"action" example:"report.approved"`
	EntityType string         `json:"entity_type" example:"daily_report"`
	EntityID   string         `json:"entity_id,omitempty"`
	Metadata   map[string]any `json:"metadata"`
}

type AuditListResponse struct {
	Events []AuditEventDTO `json:"events"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}
