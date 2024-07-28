package pool

import (
	"context"
	"sync"
)

// StaticPool represents a static pool of workers.
// Users can submit new tasks to an available worker via Submit.
type StaticPool struct {
	size int

	wg        sync.WaitGroup
	tasks     chan func(ctx context.Context)
	cancelled <-chan struct{}
}

func StartNewStaticPool(ctx context.Context, size int) *StaticPool {
	p := NewStaticPool(size)
	p.Start(ctx)

	return p
}

func NewStaticPool(size int) *StaticPool {
	return &StaticPool{
		size:  size,
		tasks: make(chan func(ctx context.Context)),
	}
}

// Start the workers in the pool (should be called only once).
func (sp *StaticPool) Start(ctx context.Context) {
	sp.cancelled = ctx.Done()
	sp.wg.Add(sp.size)
	for i := 0; i < sp.size; i++ {
		go sp.worker(ctx)
	}
}

// Submit a new task to the worker pool. If a worker is available, it will handle the task immediately;
// Otherwise, Submit will block until a worker becomes available.
func (sp *StaticPool) Submit(task func(ctx context.Context)) {
	select {
	case sp.tasks <- task:
	case <-sp.cancelled:
	}
}

// Close the pool.
// Returns an optional wait function to wait for inflight tasks.
// note: a call to Submit after Close will result in a panic.
func (sp *StaticPool) Close() (wait func()) {
	close(sp.tasks)
	return sp.wg.Wait
}

func (sp *StaticPool) worker(ctx context.Context) {
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
