package domain

import (
	"errors"
	"strings"
	"time"
)

// AuditEvent is one entry in the append-only audit log.
type AuditEvent struct {
	// ID is set on events read back from the log; zero for a new one.
	ID         int64
	OccurredAt time.Time
	// ActorID is the user who acted; empty for system events.
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
}

// Validate checks the rules every audit event must meet before it is stored.
func (e AuditEvent) Validate() error {
	switch {
	case strings.TrimSpace(e.Action) == "":
		return errors.New("audit event needs an action")
	case strings.TrimSpace(e.EntityType) == "":
		return errors.New("audit event needs an entity type")
	case e.OccurredAt.IsZero():
		return errors.New("audit event needs a time")
	}
	return nil
}
