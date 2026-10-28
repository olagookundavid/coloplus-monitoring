// Package platform holds small, product-neutral runtime helpers: the batcher,
// the worker pool and the clock.
package platform

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// OverflowPolicy says what Add does when the queue is full.
type OverflowPolicy int

const (
	// OverflowDrop discards the item and counts it.
	OverflowDrop OverflowPolicy = iota
	// OverflowBlock waits up to BlockTimeout for room, then drops and counts.
	OverflowBlock
)

// FlushFunc writes one batch. It must be safe to call from a single goroutine.
type FlushFunc[T any] func(ctx context.Context, items []T) error

type BatcherConfig struct {
	Name     string
	MaxItems int
	Interval time.Duration
	// QueueSize bounds how many items may wait for the flusher.
	QueueSize    int
	Overflow     OverflowPolicy
	BlockTimeout time.Duration
	// FlushTimeout bounds each periodic flush. Defaults to 10s.
	FlushTimeout time.Duration
}

// Batcher collects items and flushes them when MaxItems have gathered or
// Interval has passed, whichever comes first. Use it only for high-volume,
// low-criticality writes; never for money, stock or auth.
type Batcher[T any] struct {
	cfg   BatcherConfig
	flush FlushFunc[T]
	log   *slog.Logger

	in   chan T
	done chan struct{}

	mu     sync.RWMutex
	closed bool

	finalCtx context.Context

	dropped atomic.Int64
	failed  atomic.Int64
}

func NewBatcher[T any](cfg BatcherConfig, flush FlushFunc[T], log *slog.Logger) (*Batcher[T], error) {
	switch {
	case cfg.MaxItems < 1:
		return nil, errors.New("batcher needs MaxItems of at least 1")
	case cfg.Interval <= 0:
		return nil, errors.New("batcher needs a positive Interval")
	case cfg.QueueSize < 1:
		return nil, errors.New("batcher needs a QueueSize of at least 1")
	}
	if cfg.FlushTimeout <= 0 {
		cfg.FlushTimeout = 10 * time.Second
	}
	b := &Batcher[T]{
		cfg:   cfg,
		flush: flush,
		log:   log,
		in:    make(chan T, cfg.QueueSize),
		done:  make(chan struct{}),
	}
	go b.run()
	return b, nil
}

// Add queues an item. It reports false when the item was not accepted because
// the batcher is closed or the queue overflowed.
func (b *Batcher[T]) Add(item T) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return false
	}

	select {
	case b.in <- item:
		return true
	default:
	}

	if b.cfg.Overflow == OverflowBlock && b.cfg.BlockTimeout > 0 {
		timer := time.NewTimer(b.cfg.BlockTimeout)
		defer timer.Stop()
		select {
		case b.in <- item:
			return true
		case <-timer.C:
		}
	}

	b.dropped.Add(1)
	b.log.Warn("batcher queue full, item dropped", "batcher", b.cfg.Name)
	return false
}

// Close stops accepting items, flushes whatever is queued using ctx, and
// returns once the flusher has finished. Call it before the database closes.
func (b *Batcher[T]) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		<-b.done
		return nil
	}
	b.closed = true
	b.finalCtx = ctx
	close(b.in)
	b.mu.Unlock()

	select {
	case <-b.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dropped is the number of items discarded because the queue was full.
func (b *Batcher[T]) Dropped() int64 { return b.dropped.Load() }

// Failed is the number of items in batches whose flush returned an error.
func (b *Batcher[T]) Failed() int64 { return b.failed.Load() }

func (b *Batcher[T]) run() {
	defer close(b.done)

	buf := make([]T, 0, b.cfg.MaxItems)
	ticker := time.NewTicker(b.cfg.Interval)
	defer ticker.Stop()

	flush := func(ctx context.Context) {
		if len(buf) == 0 {
			return
		}
		if err := b.flush(ctx, buf); err != nil {
			b.failed.Add(int64(len(buf)))
			b.log.Error("batch flush failed", "batcher", b.cfg.Name, "items", len(buf), "error", err)
		}
		buf = buf[:0]
	}
	periodic := func() {
		ctx, cancel := context.WithTimeout(context.Background(), b.cfg.FlushTimeout)
		defer cancel()
		flush(ctx)
	}

	for {
		select {
		case item, ok := <-b.in:
			if !ok {
				flush(b.finalCtx)
				return
			}
			buf = append(buf, item)
			if len(buf) >= b.cfg.MaxItems {
				periodic()
			}
		case <-ticker.C:
			periodic()
		}
	}
}
