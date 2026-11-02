package platform

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Memo answers the same question once however many callers ask at the same
// moment, and can keep an answer for a short while. It is for read-heavy views
// that are costly to compute and where a few seconds of age do not matter.
//
// The key must carry everything that changes the answer, the caller's access
// included; Memo knows nothing about who is asking.
type Memo struct {
	clock   interface{ Now() time.Time }
	ttl     time.Duration
	timeout time.Duration
	max     int

	flight singleflight.Group
	mu     sync.Mutex
	cache  map[string]memoEntry
}

type memoEntry struct {
	at  time.Time
	val any
}

// NewMemo keeps answers for ttl (0 keeps none, but concurrent identical calls
// are still shared). A computation is cut off after timeout, and at most max
// answers are held.
func NewMemo(clock interface{ Now() time.Time }, ttl, timeout time.Duration, max int) *Memo {
	return &Memo{clock: clock, ttl: ttl, timeout: timeout, max: max, cache: map[string]memoEntry{}}
}

func (m *Memo) lookup(key string) (any, bool) {
	if m.ttl <= 0 {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.cache[key]
	if !ok || m.clock.Now().Sub(e.at) >= m.ttl {
		return nil, false
	}
	return e.val, true
}

func (m *Memo) store(key string, val any) {
	if m.ttl <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.clock.Now()
	if len(m.cache) >= m.max {
		for k, e := range m.cache {
			if now.Sub(e.at) >= m.ttl {
				delete(m.cache, k)
			}
		}
		if len(m.cache) >= m.max {
			m.cache = map[string]memoEntry{}
		}
	}
	m.cache[key] = memoEntry{at: now, val: val}
}

// Do returns a kept answer, or computes it once for everyone waiting. The
// computation is not tied to the first caller's context, so one person closing
// their browser does not fail the others waiting on the same answer; each
// caller still stops waiting when its own context ends. The value is shared:
// callers must treat it as read-only.
func Do[T any](ctx context.Context, m *Memo, key string, compute func(context.Context) (T, error)) (T, error) {
	var zero T
	if v, ok := m.lookup(key); ok {
		return v.(T), nil
	}
	ch := m.flight.DoChan(key, func() (any, error) {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.timeout)
		defer cancel()
		v, err := compute(cctx)
		if err == nil {
			m.store(key, v)
		}
		return v, err
	})
	select {
	case r := <-ch:
		if r.Err != nil {
			return zero, r.Err
		}
		return r.Val.(T), nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}
