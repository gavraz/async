package pool

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
)

func TestStaticPool_Submit(t *testing.T) {
	p := StartNewStatic(context.Background(), 1)
	done := make(chan struct{})
	p.Submit(func(ctx context.Context) {
		close(done)
	})
	<-done
}

func TestStaticPool_Close(t *testing.T) {
	t.Run("closing and waiting should complete execution for all tasks", func(t *testing.T) {
		p := StartNewStatic(context.Background(), 4)

		total := atomic.Int32{}
		for i := 0; i < 10; i++ {
			p.Submit(func(ctx context.Context) {
				total.Add(1)
			})
		}

		wait := p.Close()
		wait()
		assert.Equal(t, int32(10), total.Load())
	})

	t.Run("submit after close should panic", func(t *testing.T) {
		p := StartNewStatic(context.Background(), 4)
		p.Close()
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("panic expected")
			}
		}()
		p.Submit(func(ctx context.Context) {
			t.Errorf("should not be executed after close")
		})
	})
}
