package coalescing

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func Test_Queue(t *testing.T) {
	ctx := context.Background()

	t.Run("single call should execute", func(t *testing.T) {
		calls := 0
		execute := Queue(ctx, func(ctx context.Context) {
			calls++
		})
		<-execute(ctx)
		assert.Equal(t, 1, calls)
	})

	t.Run("two concurrent calls should execute twice", func(t *testing.T) {
		calls := 0
		blocker := make(chan struct{})
		q := newQueue(func(ctx context.Context) {
			<-blocker
			calls++
		})
		go q.start(ctx)

		waitFirst := q.enqueue(ctx)
		waitSecond := q.enqueue(ctx)

		assert.Equal(t, (<-chan struct{})(q.current.done), waitFirst)
		assert.Equal(t, (<-chan struct{})(q.next.done), waitSecond)

		close(blocker)
		<-waitFirst
		<-waitSecond

		assert.Equal(t, 2, calls)
	})

	t.Run("two consecutive calls should execute twice", func(t *testing.T) {
		calls := 0
		execute := Queue(ctx, func(ctx context.Context) {
			calls++
		})
		<-execute(ctx)
		<-execute(ctx)

		assert.Equal(t, 2, calls)
	})

	t.Run("excessive calls should be dropped", func(t *testing.T) {
		calls := 0
		blocker := make(chan struct{})
		q := newQueue(func(ctx context.Context) {
			<-blocker
			calls++
		})
		go q.start(ctx)
		execute := q.enqueue

		var done []<-chan struct{}
		done = append(done, execute(ctx), execute(ctx))

		// excessive
		done = append(done, execute(ctx), execute(ctx), execute(ctx))

		for _, d := range done[1:] {
			assert.Equal(t, (<-chan struct{})(q.next.done), d)
		}

		close(blocker)

		for _, d := range done {
			<-d
		}

		assert.Equal(t, 2, calls)
	})

	t.Run("closing the context of the queue should cancel the execution", func(t *testing.T) {
		var closedByCtx bool
		execute := Queue(ctx, func(ctx context.Context) {
			<-ctx.Done()
			closedByCtx = true
		})

		fnCtx, fnCancel := context.WithCancel(context.Background())
		fnCancel()
		<-execute(fnCtx)

		assert.True(t, closedByCtx)
	})

	t.Run("execution after the queue's context is cancelled is non-blocking and the execution will not complete", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		execute := Queue(ctx, func(ctx context.Context) {
		})
		cancel()
		execute(context.Background())
		execute(context.Background())
		done := execute(context.Background())
		select {
		case <-done:
			t.Fatalf("execution should not terminate")
		default:
		}
	})
}

func Test_stress(t *testing.T) {
	t.Skip()

	ctx := context.Background()

	calls := 0
	execute := Queue(ctx, func(ctx context.Context) {
		time.Sleep(time.Millisecond * 2)
		calls++
	})

	wg := sync.WaitGroup{}
	wg.Add(10_000)
	for i := 0; i < 10_000; i++ {
		go func() {
			<-execute(ctx)
			wg.Done()
		}()
	}

	// expected:
	// 1. no deadlock
	// 2. lower sleep interval --> lower #calls
	wg.Wait()
	fmt.Println(calls)
}
