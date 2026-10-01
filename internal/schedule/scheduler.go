package schedule

import (
	"context"
	"log/slog"
	"time"
)

// Job is a named periodic task (currencies:update, uploads:prune, recurring).
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(context.Context) error
}

// Scheduler runs jobs in-process. There is no Laravel scheduler table.
type Scheduler struct {
	jobs []Job
}

func New(jobs ...Job) *Scheduler {
	return &Scheduler{jobs: jobs}
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.jobs {
		job := job
		go s.loop(ctx, job)
	}
}

func (s *Scheduler) loop(ctx context.Context, job Job) {
	if job.Interval <= 0 {
		return
	}
	// Run once shortly after boot, then on the interval.
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := job.Run(ctx); err != nil {
				slog.Error("scheduled job failed", "job", job.Name, "err", err)
			}
			timer.Reset(job.Interval)
		}
	}
}
