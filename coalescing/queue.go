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
		// Could there be a deadlock on q.nextJob?
		//
		// The state machine flow is detailed in diagram.svg.
		// States:
		//	1. Empty - both q.current and q.next are nil
		//	2. Current only - q.current is not nil, but q.next is nil
		//	3. Full - both q.current and q.next are not nil
		// Each state specifies the length of the channel.
		// Operations:
		//	1. enqueue - a call to q.enqueue
		//	2. handle - executing the code segment under the lock while handling a job
		//	3. runtime - represents scheduling by Go's runtime
		//
		// Since all state transitions involving a lock correspond to a channel length of zero,
		// we can infer that there is no deadlock.
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
