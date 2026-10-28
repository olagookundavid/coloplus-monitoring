// Integration tests run against a real Postgres, never a mock. They are
// skipped unless TEST_DATABASE_URL is set (see `make test/integration`).
package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"coloplus-backend/internal/adapters/postgres"
	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/sql/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	pool  *pgxpool.Pool
	db    *postgres.DB
	tx    *postgres.TxManager
	audit *postgres.AuditRepository
}

func setup(t *testing.T) env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := postgres.Migrate(ctx, url, migrations.FS, log); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{URL: url, MaxConns: 4, MinConns: 0, Timezone: "Africa/Lagos"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	db := postgres.NewDB(pool)
	return env{pool: pool, db: db, tx: postgres.NewTxManager(db), audit: postgres.NewAuditRepository(db)}
}

func uniqueEntity(t *testing.T) string {
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
}

func (e env) count(t *testing.T, entityID string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE entity_id = $1`, entityID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func event(entityID, action string) domain.AuditEvent {
	return domain.AuditEvent{
		OccurredAt: time.Now().UTC(),
		ActorID:    "user-1",
		Action:     action,
		EntityType: "daily_report",
		EntityID:   entityID,
		Metadata:   map[string]any{"site": "LOS-001"},
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	e := setup(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := postgres.Migrate(context.Background(), os.Getenv("TEST_DATABASE_URL"), migrations.FS, log); err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	_ = e
}

func TestSessionTimezoneIsPinned(t *testing.T) {
	e := setup(t)
	var tz string
	if err := e.pool.QueryRow(context.Background(), `SHOW TimeZone`).Scan(&tz); err != nil {
		t.Fatal(err)
	}
	if tz != "Africa/Lagos" {
		t.Fatalf("session time zone = %q, want Africa/Lagos", tz)
	}
}

func TestAuditInsertBatchWritesEveryEvent(t *testing.T) {
	e := setup(t)
	id := uniqueEntity(t)
	events := []domain.AuditEvent{event(id, "a"), event(id, "b"), event(id, "c")}

	if err := e.audit.InsertBatch(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	if got := e.count(t, id); got != 3 {
		t.Fatalf("rows = %d, want 3", got)
	}
}

func TestAuditInsertAcceptsNilMetadata(t *testing.T) {
	e := setup(t)
	id := uniqueEntity(t)
	ev := event(id, "a")
	ev.Metadata = nil
	if err := e.audit.Insert(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if got := e.count(t, id); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
}

func TestTxManagerCommitsAndRollsBack(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	committed := uniqueEntity(t) + "-commit"
	if err := e.tx.WithTx(ctx, func(ctx context.Context) error {
		return e.audit.Insert(ctx, event(committed, "approved"))
	}); err != nil {
		t.Fatal(err)
	}
	if got := e.count(t, committed); got != 1 {
		t.Fatalf("committed rows = %d, want 1", got)
	}

	rolledBack := uniqueEntity(t) + "-rollback"
	boom := errors.New("business rule failed")
	err := e.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := e.audit.Insert(ctx, event(rolledBack, "approved")); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the business error", err)
	}
	if got := e.count(t, rolledBack); got != 0 {
		t.Fatalf("rolled-back rows = %d, want 0: the audit row must not outlive its transaction", got)
	}
}

func TestTxManagerNestedCallJoinsTheOuterTransaction(t *testing.T) {
	e := setup(t)
	id := uniqueEntity(t)
	boom := errors.New("outer failed")

	err := e.tx.WithTx(context.Background(), func(ctx context.Context) error {
		if err := e.tx.WithTx(ctx, func(ctx context.Context) error {
			return e.audit.Insert(ctx, event(id, "inner"))
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if got := e.count(t, id); got != 0 {
		t.Fatalf("rows = %d, want 0: the inner call must roll back with the outer one", got)
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	id := uniqueEntity(t)
	if err := e.audit.Insert(ctx, event(id, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE audit_log SET action = 'tampered' WHERE entity_id = $1`, id); err == nil {
		t.Fatal("UPDATE on audit_log must be rejected")
	}
	if _, err := e.pool.Exec(ctx, `DELETE FROM audit_log WHERE entity_id = $1`, id); err == nil {
		t.Fatal("DELETE on audit_log must be rejected")
	}
	if got := e.count(t, id); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
}
