package future

// Future allows a value to be set asynchronously.
// The value can be set only once using SetResult, but it can be retrieved multiple times using WaitResult.
// Callers may use Done to wait in a select statement.
type Future[T any] struct {
	done   chan struct{}
	result T
}

func New[T any]() *Future[T] {
	return &Future[T]{done: make(chan struct{})}
}

// SetResult assigns a value to the future and makes it available to waiting routines.
// The result can be set only once.
func (f *Future[T]) SetResult(result T) {
	f.result = result
	close(f.done)
}

// Done is a channel used in select statements to wait until the result is available.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// WaitResult blocks until the result is set and then returns it.
// If the result is already set, it returns immediately.
func (f *Future[T]) WaitResult() T {
	<-f.done
	return f.result
}
