package pool

import (
	"context"
	"sync"
)

// Static represents a pool of workers with a fixed size.
// Tasks can be submitted using Submit which will block until a worker is available.
// The pool can be closed using Close or canceled by the ctx.
type Static struct {
	size int

	wg        sync.WaitGroup
	tasks     chan func(ctx context.Context)
	cancelled <-chan struct{}
}

func StartNewStatic(ctx context.Context, size int) *Static {
	start := NewStaticStarter(size)
	return start(ctx)
}

func NewStaticStarter(size int) (start func(ctx context.Context) *Static) {
	s := &Static{
		size:  size,
		tasks: make(chan func(ctx context.Context)),
	}

	return func(ctx context.Context) *Static {
		s.Start(ctx)
		return s
	}
}

// Start the workers in the pool (should be called only once).
func (sp *Static) Start(ctx context.Context) {
	sp.cancelled = ctx.Done()
	sp.wg.Add(sp.size)
	for i := 0; i < sp.size; i++ {
		go sp.worker(ctx)
	}
}

// Submit a new task to the worker pool. If a worker is available, it will handle the task immediately;
// Otherwise, Submit will block until a worker becomes available.
func (sp *Static) Submit(task func(ctx context.Context)) {
	select {
	case sp.tasks <- task:
	case <-sp.cancelled:
	}
}

// Close the pool.
// Returns an optional wait function to wait for inflight tasks.
// note: a call to Submit after Close will result in a panic.
func (sp *Static) Close() (wait func()) {
	close(sp.tasks)
	return sp.wg.Wait
}

func (sp *Static) worker(ctx context.Context) {
	defer sp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-sp.tasks:
			if !ok {
				return
			}
			task(ctx)
		}
	}
}
