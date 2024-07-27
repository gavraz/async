package future

type Future[T any] struct {
	done   chan struct{}
	result T
}

func New[T any]() *Future[T] {
	return &Future[T]{done: make(chan struct{})}
}

func (f *Future[T]) SetResult(result T) {
	f.result = result
	close(f.done)
}

func (f *Future[T]) Wait() {
	<-f.done
}

func (f *Future[T]) WaitResult() T {
	<-f.done
	return f.result
}
