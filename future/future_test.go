package future

import (
	"testing"
)

func Test_PromiseFuture(t *testing.T) {
	t.Run("Setting the value before retrieving the future returns the promised value", func(t *testing.T) {
		p := NewPromise[int]()
		p.Set(42)
		result := p.Future().Value()
		if result != 42 {
			t.Errorf("Expected result to be 42, got %d", result)
		}
	})

	t.Run("Setting the value after retrieving the future returns the promised value", func(t *testing.T) {
		p := NewPromise[int]()
		f := p.Future()
		p.Set(42)
		result := f.Value()
		if result != 42 {
			t.Errorf("Expected result to be 42, got %d", result)
		}
	})

	t.Run("Done blocks until the value is set", func(t *testing.T) {
		p := NewPromise[int]()
		f := p.Future()
		select {
		case <-f.Done():
			t.Error("done should block until value is set")
		default:
		}

		p.Set(99)
		select {
		case <-f.Done():
		default:
			t.Error("done should unblock after the value is set")
		}
	})

	t.Run("The value can be set only once", func(t *testing.T) {
		p := NewPromise[int]()
		p.Set(1)
		defer func() {
			if r := recover(); r == nil {
				t.Error("second call to SetResult did not panic")
			}
		}()
		p.Set(2)
	})
}

func TestFutureConcurrency(t *testing.T) {
	p := NewPromise[int]()
	f := p.Future()
	for i := 0; i < 100; i++ {
		go func() {
			result := f.Value()
			if result != 10 {
				t.Logf("Got result: %d", result)
			}
		}()
	}

	go p.Set(10)

	<-f.Done()
}
