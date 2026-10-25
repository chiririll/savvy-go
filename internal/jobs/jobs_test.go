package jobs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueRunsAllJobsBeforeShutdown(t *testing.T) {
	q := NewQueue(3)
	var done atomic.Int32
	for range 50 {
		q.Enqueue(func(context.Context) { done.Add(1) })
	}
	q.Shutdown(context.Background())
	if got := done.Load(); got != 50 {
		t.Errorf("%d of 50 jobs ran", got)
	}
}

func TestQueueSurvivesAPanickingJob(t *testing.T) {
	q := NewQueue(1) // one worker: if the panic killed it, the next job would never run
	ran := make(chan struct{})
	q.Enqueue(func(context.Context) { panic("boom") })
	q.Enqueue(func(context.Context) { close(ran) })
	select {
	case <-ran:
	case <-time.After(5 * time.Second):
		t.Fatal("the worker died with the panicking job")
	}
	q.Shutdown(context.Background())
}

func TestQueueWithoutWorkersStillGetsSome(t *testing.T) {
	q := NewQueue(0)
	var wg sync.WaitGroup
	wg.Add(1)
	q.Enqueue(func(context.Context) { wg.Done() })
	wg.Wait()
	q.Shutdown(context.Background())
}

func TestShutdownGivesUpWhenContextEnds(t *testing.T) {
	q := NewQueue(1)
	release := make(chan struct{})
	q.Enqueue(func(context.Context) { <-release })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	finished := make(chan struct{})
	go func() { q.Shutdown(ctx); close(finished) }()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown waited past its context")
	}
	close(release)
}

func TestSchedulerRunsJobsIntervalSurvivesFailures(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var failing atomic.Int32
	reached := make(chan struct{})
	quick(NewScheduler(Job{Name: "flaky", Interval: 20 * time.Millisecond, Run: func(context.Context) error {
		if failing.Add(1) == 3 {
			close(reached)
		}
		return errors.New("always fails")
	}})).Start(ctx)

	select {
	case <-reached:
	case <-time.After(10 * time.Second):
		t.Fatalf("a failing job must keep running; it ran %d times", failing.Load())
	}
}

func TestSchedulerStopsWithItsContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	var runs atomic.Int32
	quick(NewScheduler(Job{Name: "j", Interval: 10 * time.Millisecond, Run: func(context.Context) error {
		runs.Add(1)
		return nil
	}})).Start(ctx)
	cancel() // before the first run
	time.Sleep(100 * time.Millisecond)
	if got := runs.Load(); got != 0 {
		t.Errorf("a cancelled scheduler ran its job %d times", got)
	}
}

func TestSchedulerIgnoresJobsWithoutAnInterval(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var runs atomic.Int32
	quick(NewScheduler(Job{Name: "never", Run: func(context.Context) error { runs.Add(1); return nil }})).Start(ctx)
	time.Sleep(100 * time.Millisecond)
	if got := runs.Load(); got != 0 {
		t.Errorf("a job without an interval ran %d times", got)
	}
}

// quick shortens the delay before the first run, which tests have no reason to wait for.
func quick(s *Scheduler) *Scheduler {
	s.firstRun = 50 * time.Millisecond
	return s
}
