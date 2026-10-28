package unit

import (
	"context"
	"testing"
	"time"

	"coloplus-backend/internal/platform"
)

func newTestBatcher(t *testing.T, cfg platform.BatcherConfig, flush platform.FlushFunc[int]) *platform.Batcher[int] {
	t.Helper()
	b, err := platform.NewBatcher(cfg, flush, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBatcher_FlushesWhenSizeReached(t *testing.T) {
	var rec recorder[int]
	b := newTestBatcher(t, platform.BatcherConfig{MaxItems: 3, Interval: time.Hour, QueueSize: 10},
		func(_ context.Context, items []int) error { rec.add(items); return nil })
	t.Cleanup(func() { _ = b.Close(context.Background()) })

	for i := 1; i <= 3; i++ {
		b.Add(i)
	}
	if !eventually(func() bool { return rec.total() == 3 }) {
		t.Fatalf("batches = %v, want one batch of 3", rec.snapshot())
	}
	if got := len(rec.snapshot()); got != 1 {
		t.Fatalf("flushed %d batches, want 1", got)
	}
}

func TestBatcher_CloseFlushesRemainder(t *testing.T) {
	var rec recorder[int]
	b := newTestBatcher(t, platform.BatcherConfig{MaxItems: 100, Interval: time.Hour, QueueSize: 10},
		func(_ context.Context, items []int) error { rec.add(items); return nil })

	b.Add(1)
	b.Add(2)
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rec.total() != 2 {
		t.Fatalf("flushed %d items on close, want 2", rec.total())
	}
	if b.Add(3) {
		t.Fatal("Add after Close must be refused")
	}
	if err := b.Close(context.Background()); err != nil {
		t.Fatalf("second Close should be a no-op, got %v", err)
	}
}

func TestBatcher_DropsWhenQueueFull(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	b := newTestBatcher(t, platform.BatcherConfig{MaxItems: 1, Interval: time.Hour, QueueSize: 1, Overflow: platform.OverflowDrop},
		func(_ context.Context, _ []int) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return nil
		})

	b.Add(1) // taken by the flusher, which then blocks
	<-started
	if !b.Add(2) { // fills the queue
		t.Fatal("second item should fit in the queue")
	}
	if b.Add(3) { // queue full
		t.Fatal("third item should be dropped")
	}
	if b.Dropped() != 1 {
		t.Fatalf("dropped = %d, want 1", b.Dropped())
	}
	close(release)
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
