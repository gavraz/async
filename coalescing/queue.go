package coalescing

import (
	"context"
	"sync"
)

type job struct {
	ctx  context.Context
	done chan struct{}
}

type queue struct {
	fn      func(ctx context.Context)
	nextJob chan *job

	mu      sync.Mutex
	current *job
	next    *job
}

func (q *queue) start(ctx context.Context) {
	for {
		select {
		case j := <-q.nextJob:
			q.fn(j.ctx)
			close(j.done)

			q.mu.Lock()
			if q.next != nil {
				q.current = q.next
				q.next = nil
				q.nextJob <- q.current
			} else {
				q.current = nil
			}
			q.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (q *queue) enqueue(ctx context.Context) <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	if empty := q.current == nil; empty {
		q.current = &job{ctx: ctx, done: make(chan struct{})}
		q.nextJob <- q.current
		return q.current.done
	}

	if excessive := q.next != nil; excessive {
		return q.next.done
	}

	q.next = &job{ctx: ctx, done: make(chan struct{})}
	return q.next.done
}

func newQueue(fn func(ctx context.Context)) *queue {
	return &queue{
		nextJob: make(chan *job, 1),
		fn:      fn,
	}
}

// Queue is used for coalescing repeated executions of the same function (fn).
// it returns a non-blocking execute function to be called for every execution request.
// the execution function returns an optional done channel to await completion.
//
// Guarantees:
// - If the first call to execute is yet to complete, the second will be executed right after the first completes.
// - If two calls are to be executed but yet to complete, any subsequent execution will be discarded.
// - Discarded executions complete upon completion of the second queue.
func Queue(ctx context.Context, fn func(ctx context.Context)) (execute func(ctx context.Context) (done <-chan struct{})) {
	q := newQueue(fn)
	go q.start(ctx)

	return q.enqueue
}
