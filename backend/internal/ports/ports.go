// Package ports holds the interfaces services depend on. Adapters implement
// them; nothing here imports an adapter, a driver or a vendor SDK.
package ports

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
)

// Clock is injected wherever the current time matters, so rules can be tested.
type Clock interface {
	Now() time.Time
}

// TxManager runs fn inside one database transaction. The context handed to fn
// carries the transaction, so repositories called with it join the same one.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AuditStore persists audit events. Insert joins a transaction carried by the
// context; InsertBatch is the bulk path used by the batcher.
type AuditStore interface {
	Insert(ctx context.Context, event domain.AuditEvent) error
	InsertBatch(ctx context.Context, events []domain.AuditEvent) error
}

// AuditQueue buffers audit events for batched writes. Add reports false when
// the event was not accepted.
type AuditQueue interface {
	Add(event domain.AuditEvent) bool
}

// WorkerPool runs background tasks (email sends, exports) off the request path.
type WorkerPool interface {
	Submit(task func()) error
}

// Pinger checks that a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}
