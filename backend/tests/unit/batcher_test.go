package unit

import (
	"context"
	"errors"
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

func TestBatcher_FlushesWhenIntervalPasses(t *testing.T) {
	var rec recorder[int]
	b := newTestBatcher(t, platform.BatcherConfig{MaxItems: 100, Interval: 30 * time.Millisecond, QueueSize: 10},
		func(_ context.Context, items []int) error { rec.add(items); return nil })
	t.Cleanup(func() { _ = b.Close(context.Background()) })

	b.Add(42)
	if !eventually(func() bool { return rec.total() == 1 }) {
		t.Fatal("item was not flushed by the interval")
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

func TestBatcher_BlockPolicyWaitsThenDrops(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	b := newTestBatcher(t, platform.BatcherConfig{
		MaxItems: 1, Interval: time.Hour, QueueSize: 1,
		Overflow: platform.OverflowBlock, BlockTimeout: 20 * time.Millisecond,
	}, func(_ context.Context, _ []int) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return nil
	})

	b.Add(1)
	<-started
	b.Add(2)
	began := time.Now()
	if b.Add(3) {
		t.Fatal("third item should be dropped after the wait")
	}
	if time.Since(began) < 15*time.Millisecond {
		t.Fatal("Add should have waited for room before dropping")
	}
	close(release)
	_ = b.Close(context.Background())
}

func TestBatcher_FlushErrorIsCountedAndLaterBatchesStillFlush(t *testing.T) {
	var rec recorder[int]
	calls := 0
	b := newTestBatcher(t, platform.BatcherConfig{MaxItems: 1, Interval: time.Hour, QueueSize: 10},
		func(_ context.Context, items []int) error {
			calls++
			if calls == 1 {
				return errors.New("database down")
			}
			rec.add(items)
			return nil
		})

	b.Add(1)
	b.Add(2)
	if err := b.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b.Failed() != 1 {
		t.Fatalf("failed = %d, want 1", b.Failed())
	}
	if rec.total() != 1 {
		t.Fatalf("second batch was not flushed: %v", rec.snapshot())
	}
}

func TestNewBatcher_RejectsBadConfig(t *testing.T) {
	flush := func(context.Context, []int) error { return nil }
	for name, cfg := range map[string]platform.BatcherConfig{
		"no size":     {MaxItems: 0, Interval: time.Second, QueueSize: 1},
		"no interval": {MaxItems: 1, Interval: 0, QueueSize: 1},
		"no queue":    {MaxItems: 1, Interval: time.Second, QueueSize: 0},
	} {
		if _, err := platform.NewBatcher(cfg, flush, discardLogger()); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
