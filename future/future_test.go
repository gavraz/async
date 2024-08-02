package future

import (
	"testing"
)

func TestFuture(t *testing.T) {
	t.Run("WaitResult returns the correct result", func(t *testing.T) {
		f := New[int]()
		f.SetResult(42)
		result := f.WaitResult()
		if result != 42 {
			t.Errorf("Expected result to be 42, got %d", result)
		}
	})

	t.Run("Wait blocks until SetResult is called", func(t *testing.T) {
		f := New[int]()
		f.SetResult(99)
		select {
		case <-f.done:
			// Expected behavior
		default:
			t.Error("Wait should unblock after SetResult is called")
		}
	})

	t.Run("SetResult can only set the result once", func(t *testing.T) {
		future := New[int]()
		future.SetResult(1)
		defer func() {
			if r := recover(); r == nil {
				t.Error("second call to SetResult did not panic")
			}
		}()
		future.SetResult(2)
	})
}

func TestFutureConcurrency(t *testing.T) {
	f := New[int]()

	for i := 0; i < 100; i++ {
		go func() {
			result := f.WaitResult()
			if result != 10 {
				t.Logf("Got result: %d", result)
			}
		}()
	}

	go f.SetResult(10)

	<-f.Done()
}
