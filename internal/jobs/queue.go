package jobs

import (
	"context"
	"log/slog"
	"sync"
)

// Queue is an in-process worker pool. Laravel's queue/cache sqlite files
// are not used; work stays in this process.
type Queue struct {
	jobs chan func(context.Context)
	wg   sync.WaitGroup
}

func New(workers int) *Queue {
	if workers < 1 {
		workers = 2
	}
	q := &Queue{jobs: make(chan func(context.Context), 64)}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.loop()
	}
	return q
}

func (q *Queue) Enqueue(fn func(context.Context)) {
	q.jobs <- fn
}

func (q *Queue) Shutdown(ctx context.Context) {
	close(q.jobs)
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		slog.Warn("job queue shutdown interrupted")
	}
}

func (q *Queue) loop() {
	defer q.wg.Done()
	for fn := range q.jobs {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("job panic", "recover", rec)
				}
			}()
			fn(context.Background())
		}()
	}
}
