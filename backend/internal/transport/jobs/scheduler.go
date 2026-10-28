// Package jobs wires scheduled work. Jobs only call services.
package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Job is one unit of scheduled work.
type Job func(ctx context.Context) error

// Scheduler runs jobs on standard five-field cron specs.
type Scheduler struct {
	cron   *cron.Cron
	ctx    context.Context
	cancel context.CancelFunc
	log    *slog.Logger
}

func NewScheduler(log *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{cron: cron.New(), ctx: ctx, cancel: cancel, log: log}
}

// Add registers a job. A bad spec is returned so the caller can fail at boot.
func (s *Scheduler) Add(name, spec string, job Job) error {
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return fmt.Errorf("job %q has an invalid cron spec %q: %w", name, spec, err)
	}
	s.cron.Schedule(schedule, cron.FuncJob(func() { s.run(name, job) }))
	return nil
}

func (s *Scheduler) Start() { s.cron.Start() }

// Stop cancels the context shared by running jobs, then waits for them.
func (s *Scheduler) Stop() {
	s.cancel()
	<-s.cron.Stop().Done()
}

func (s *Scheduler) run(name string, job Job) {
	if s.ctx.Err() != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("scheduled job panicked", "job", name, "panic", r)
		}
	}()
	if err := job(s.ctx); err != nil && s.ctx.Err() == nil {
		s.log.Error("scheduled job failed", "job", name, "error", err)
	}
}
