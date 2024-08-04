package future

// Promise is used to commit for value.
// The promised value can be set using Set and be retrieved using Future.
type Promise[T any] struct {
	done   chan struct{}
	result T
}

func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{done: make(chan struct{})}
}

// Set assigns a value to the future and makes it available to waiting routines.
// The result can be set only once.
func (p *Promise[T]) Set(result T) {
	p.result = result
	close(p.done)
}

func (p *Promise[T]) Future() *Future[T] {
	return &Future[T]{
		p: p,
	}
}

// Future is used to wait for a value that may be available in the future.
// Callers may use Done to wait in a select statement.
type Future[T any] struct {
	p *Promise[T]
}

// Done is a channel used in select statements to wait until the value is available.
func (f *Future[T]) Done() <-chan struct{} {
	return f.p.done
}

// Value blocks until the value is set and then returns it.
// If the result is already set, it returns immediately.
func (f *Future[T]) Value() T {
	<-f.p.done
	return f.p.result
}
