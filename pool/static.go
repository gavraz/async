package pool

import (
	"context"
	"sync"
)

type StaticPool struct {
	wg    sync.WaitGroup
	tasks chan func(ctx context.Context)
}

func NewStaticPool() *StaticPool {
	return &StaticPool{
		tasks: make(chan func(ctx context.Context)),
	}
}

func StartNewStaticPool(ctx context.Context, size int) *StaticPool {
	p := &StaticPool{
		tasks: make(chan func(ctx context.Context)),
	}

	p.start(ctx, size)

	return p
}

func (sp *StaticPool) Submit(task func(ctx context.Context)) {
	sp.tasks <- task
}

func (sp *StaticPool) Close() func() {
	close(sp.tasks)
	return sp.wg.Wait
}

func (sp *StaticPool) start(ctx context.Context, size int) {
	sp.wg.Add(size)
	for i := 0; i < size; i++ {
		go sp.worker(ctx)
	}
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
