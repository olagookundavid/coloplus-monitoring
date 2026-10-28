package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
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

func TestAuditService_RecordQueuesAndStampsTime(t *testing.T) {
	store, queue := &fakeAuditStore{}, &fakeAuditQueue{accept: true}
	svc := newAuditService(store, queue)

	if err := svc.Record(context.Background(), domain.AuditEvent{Action: "report.submitted", EntityType: "daily_report"}); err != nil {
		t.Fatal(err)
	}
	if len(queue.queued) != 1 || !queue.queued[0].OccurredAt.Equal(testNow) {
		t.Fatalf("queued = %+v, want one event stamped %v", queue.queued, testNow)
	}
	if len(store.inserted) != 0 {
		t.Fatal("a batched event must not be written immediately")
	}
}

func TestAuditService_RecordNeverFailsTheCallerWhenQueueIsFull(t *testing.T) {
	svc := newAuditService(&fakeAuditStore{}, &fakeAuditQueue{accept: false})
	if err := svc.Record(context.Background(), domain.AuditEvent{Action: "a", EntityType: "b"}); err != nil {
		t.Fatalf("a full queue must not fail the business action: %v", err)
	}
}

func TestAuditService_RecordRejectsInvalidEvent(t *testing.T) {
	svc := newAuditService(&fakeAuditStore{}, &fakeAuditQueue{accept: true})
	err := svc.Record(context.Background(), domain.AuditEvent{EntityType: "daily_report"})
	if !errors.Is(err, apperrors.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestAuditService_RecordInTxWritesImmediately(t *testing.T) {
	store, queue := &fakeAuditStore{}, &fakeAuditQueue{accept: true}
	svc := newAuditService(store, queue)

	if err := svc.RecordInTx(context.Background(), domain.AuditEvent{Action: "report.approved", EntityType: "daily_report"}); err != nil {
		t.Fatal(err)
	}
	if len(store.inserted) != 1 || len(queue.queued) != 0 {
		t.Fatalf("inserted=%d queued=%d, want 1 and 0", len(store.inserted), len(queue.queued))
	}
}

func TestAuditService_RecordInTxSurfacesStoreFailure(t *testing.T) {
	store := &fakeAuditStore{err: errors.New("connection reset")}
	svc := newAuditService(store, &fakeAuditQueue{accept: true})

	if err := svc.RecordInTx(context.Background(), domain.AuditEvent{Action: "a", EntityType: "b"}); err == nil {
		t.Fatal("a failed transactional audit write must fail the transaction")
	}
}
