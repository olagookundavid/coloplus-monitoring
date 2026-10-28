package unit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"coloplus-backend/internal/platform"
)

func TestWorkerPool_RunsTasks(t *testing.T) {
	p, err := platform.NewWorkerPool(platform.WorkerPoolConfig{Size: 2, QueueSize: 4}, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	var n atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		if err := p.Submit(func() { defer wg.Done(); n.Add(1) }); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	if n.Load() != 5 {
		t.Fatalf("ran %d tasks, want 5", n.Load())
	}
}

func TestWorkerPool_SurvivesPanickingTask(t *testing.T) {
	p, _ := platform.NewWorkerPool(platform.WorkerPoolConfig{Size: 1, QueueSize: 1}, discardLogger())
	_ = p.Submit(func() { panic("boom") })

	done := make(chan struct{})
	if !eventually(func() bool {
		return p.Submit(func() { close(done) }) == nil
	}) {
		t.Fatal("pool stopped accepting tasks after a panic")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task after the panic never ran")
	}
}

func TestWorkerPool_RejectsWhenFull(t *testing.T) {
	p, _ := platform.NewWorkerPool(platform.WorkerPoolConfig{Size: 1, QueueSize: 0}, discardLogger())
	block := make(chan struct{})
	if err := p.Submit(func() { <-block }); err != nil {
		t.Fatal(err)
	}
	if err := p.Submit(func() {}); !errors.Is(err, platform.ErrPoolOverloaded) {
		t.Fatalf("err = %v, want ErrPoolOverloaded", err)
	}
	close(block)
}

func TestWorkerPool_DrainWaitsForRunningTasks(t *testing.T) {
	p, _ := platform.NewWorkerPool(platform.WorkerPoolConfig{Size: 1, QueueSize: 1}, discardLogger())
	var finished atomic.Bool
	_ = p.Submit(func() { time.Sleep(50 * time.Millisecond); finished.Store(true) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Drain(ctx); err != nil {
		t.Fatal(err)
	}
	if !finished.Load() {
		t.Fatal("Drain returned before the running task finished")
	}
}

func TestNewWorkerPool_RejectsZeroSize(t *testing.T) {
	if _, err := platform.NewWorkerPool(platform.WorkerPoolConfig{Size: 0}, discardLogger()); err == nil {
		t.Fatal("expected an error")
	}
}
