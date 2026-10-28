package unit

import (
	"io"
	"log/slog"
	"sync"
	"time"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// recorder collects flushed batches for assertions.
type recorder[T any] struct {
	mu      sync.Mutex
	batches [][]T
}

func (r *recorder[T]) add(items []T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, append([]T(nil), items...))
}

func (r *recorder[T]) snapshot() [][]T {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]T(nil), r.batches...)
}

func (r *recorder[T]) total() int {
	n := 0
	for _, b := range r.snapshot() {
		n += len(b)
	}
	return n
}

// eventually polls cond until it holds or the deadline passes.
func eventually(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}
