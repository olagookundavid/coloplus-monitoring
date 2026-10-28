package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/panjf2000/ants/v2"
)

// ErrPoolOverloaded is returned by Submit when the pool and its wait queue
// are both full. Callers decide whether to retry later or give up.
var ErrPoolOverloaded = errors.New("worker pool is overloaded")

type WorkerPoolConfig struct {
	// Size is how many tasks run at once.
	Size int
	// QueueSize is how many submitters may wait for a free worker before
	// Submit starts rejecting. Zero rejects as soon as every worker is busy.
	QueueSize int
}

// WorkerPool is a bounded pool of goroutines. Every task is wrapped in
// recover(), so one bad task cannot take the process down.
type WorkerPool struct {
	pool *ants.Pool
	log  *slog.Logger
}

func NewWorkerPool(cfg WorkerPoolConfig, log *slog.Logger) (*WorkerPool, error) {
	if cfg.Size < 1 {
		return nil, errors.New("worker pool needs a size of at least 1")
	}
	opts := []ants.Option{
		ants.WithPanicHandler(func(v any) { log.Error("worker task panicked", "panic", v) }),
	}
	if cfg.QueueSize > 0 {
		opts = append(opts, ants.WithMaxBlockingTasks(cfg.QueueSize))
	} else {
		// In ants a limit of 0 means "unlimited waiting", so a zero queue is
		// expressed as non-blocking: reject as soon as every worker is busy.
		opts = append(opts, ants.WithNonblocking(true))
	}
	pool, err := ants.NewPool(cfg.Size, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating worker pool: %w", err)
	}
	return &WorkerPool{pool: pool, log: log}, nil
}

// Submit runs task on a worker. It blocks briefly when all workers are busy
// and returns ErrPoolOverloaded once the wait queue is full.
func (p *WorkerPool) Submit(task func()) error {
	err := p.pool.Submit(func() {
		defer func() {
			if r := recover(); r != nil {
				p.log.Error("worker task panicked", "panic", r)
			}
		}()
		task()
	})
	switch {
	case errors.Is(err, ants.ErrPoolOverload):
		return ErrPoolOverloaded
	case err != nil:
		return fmt.Errorf("submitting task: %w", err)
	}
	return nil
}

// Drain stops accepting tasks and waits for running ones until ctx ends.
func (p *WorkerPool) Drain(ctx context.Context) error {
	timeout := time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	if err := p.pool.ReleaseTimeout(timeout); err != nil {
		return fmt.Errorf("draining worker pool: %w", err)
	}
	return nil
}
