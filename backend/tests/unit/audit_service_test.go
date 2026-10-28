package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/services"
)

type fakeAuditStore struct {
	inserted []domain.AuditEvent
	err      error
}

func (f *fakeAuditStore) Insert(_ context.Context, e domain.AuditEvent) error {
	if f.err != nil {
		return f.err
	}
	f.inserted = append(f.inserted, e)
	return nil
}

func (f *fakeAuditStore) InsertBatch(_ context.Context, es []domain.AuditEvent) error {
	f.inserted = append(f.inserted, es...)
	return nil
}

type fakeAuditQueue struct {
	accept bool
	queued []domain.AuditEvent
}

func (f *fakeAuditQueue) Add(e domain.AuditEvent) bool {
	if !f.accept {
		return false
	}
	f.queued = append(f.queued, e)
	return true
}

var testNow = time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)

func newAuditService(store *fakeAuditStore, queue *fakeAuditQueue) *services.AuditService {
	return services.NewAuditService(store, queue, fixedClock{testNow}, discardLogger())
}

func TestAuditService_RecordNeverFailsTheCallerWhenQueueIsFull(t *testing.T) {
	svc := newAuditService(&fakeAuditStore{}, &fakeAuditQueue{accept: false})
	if err := svc.Record(context.Background(), domain.AuditEvent{Action: "a", EntityType: "b"}); err != nil {
		t.Fatalf("a full queue must not fail the business action: %v", err)
	}
}

func TestAuditService_RecordInTxSurfacesStoreFailure(t *testing.T) {
	store := &fakeAuditStore{err: errors.New("connection reset")}
	svc := newAuditService(store, &fakeAuditQueue{accept: true})

	if err := svc.RecordInTx(context.Background(), domain.AuditEvent{Action: "a", EntityType: "b"}); err == nil {
		t.Fatal("a failed transactional audit write must fail the transaction")
	}
}
