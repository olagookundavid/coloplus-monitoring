package unit

import (
	"context"
	"testing"

	"coloplus-backend/internal/transport/jobs"
)

func TestScheduler_AcceptsOnlyFiveFieldSpecs(t *testing.T) {
	s := jobs.NewScheduler(discardLogger())
	job := func(context.Context) error { return nil }

	if err := s.Add("ok", "*/5 * * * *", job); err != nil {
		t.Fatalf("a standard five-field spec was rejected: %v", err)
	}
	// A six-field spec has seconds. Treating it as valid once made a job run
	// every five seconds and exhausted the connection pool in production.
	if err := s.Add("six-fields", "*/5 * * * * *", job); err == nil {
		t.Fatal("a six-field spec must be rejected")
	}
	if err := s.Add("garbage", "not a spec", job); err == nil {
		t.Fatal("a bad spec must be rejected")
	}
}
