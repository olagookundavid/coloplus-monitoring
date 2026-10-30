package unit

import (
	"errors"
	"testing"
	"time"

	"coloplus-backend/internal/platform"
)

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
