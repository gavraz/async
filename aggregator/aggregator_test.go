package aggregator

import (
	"context"
	"github.com/coder/quartz"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestAggregator_NewStarter(t *testing.T) {
	starter := NewStarter[int](Config[int]{
		MaxDuration: DisableTimeLimit,
		MaxCount:    ImmediateDelivery,
	}, quartz.NewMock(t))

	ctx, cancel := context.WithCancel(context.Background())
	a := starter(ctx)
	a.conf.Handler = func(events []int) {
		a.pendingHandling <- events
	}
	a.conf.OnQueueFull = func(events []int) {
		a.pendingHandling <- events
	}
	cancel()

	// this should be non-blocking since ctx is done
	a.OnEvent(1)
	a.OnEvent(1)
}

func TestAggregator_ZeroDurationShouldPanicOnInitialization(t *testing.T) {
	defer func() {
		r := recover()
		assert.Equal(t, "bad config: max duration cannot be zero", r)
	}()
	_ = NewStarter(Config[int]{
		MaxDuration: 0,
	}, nil)
}

func TestAggregator_OnEvent(t *testing.T) {
	ctx := context.Background()
	actionComplete := make(chan struct{})
	clock := quartz.NewMock(t)
	total := 0
	a := startNew[int](ctx, Config[int]{
		MaxDuration: time.Nanosecond,
		MaxCount:    3,
		Handler: func(events []int) {
			total += len(events)
			actionComplete <- struct{}{}
		},
		QueueSize: 1,
		OnQueueFull: func([]int) {
			t.Fatalf("unexpected buffer full")
		},
	}, clock)

	t.Run("if num of buffered events is lower than max count then events should be sent only after duration", func(t *testing.T) {
		a.OnEvent(1)
		assert.Equal(t, 0, total)
		assert.Equal(t, 1, len(a.bufferedEvents))
		a.OnEvent(2)
		assert.Equal(t, 0, total)
		assert.Equal(t, 2, len(a.bufferedEvents))
		select {
		case <-a.pendingHandling:
			t.Fatalf("unexpected handle of events")
		default:
		}
		clock.Advance(a.conf.MaxDuration).MustWait(ctx)
		<-actionComplete
		assert.Equal(t, 2, total)
		assert.Equal(t, 0, len(a.bufferedEvents))
	})

	t.Run("if num of buffered events is higher than max count then events should be flushed", func(t *testing.T) {
		total = 0
		a.OnEvent(1)
		assert.Equal(t, 0, total)
		assert.Equal(t, 1, len(a.bufferedEvents))
		a.OnEvent(2)
		assert.Equal(t, 0, total)
		assert.Equal(t, 2, len(a.bufferedEvents))
		a.OnEvent(3)
		<-actionComplete
		assert.Equal(t, 3, total)
		assert.Equal(t, 0, len(a.bufferedEvents))
	})

	t.Run("consecutive max duration with withstanding events should be flushed", func(t *testing.T) {
		total = 0

		// first with standing event
		a.OnEvent(1)
		assert.Equal(t, 0, total)
		assert.Equal(t, 1, len(a.bufferedEvents))
		select {
		case <-a.pendingHandling:
			t.Fatalf("unexpected handle of events")
		default:
		}
		clock.Advance(a.conf.MaxDuration).MustWait(ctx)
		<-actionComplete

		// second withstanding event
		a.OnEvent(1)
		assert.Equal(t, 1, total)
		assert.Equal(t, 1, len(a.bufferedEvents))
		select {
		case <-a.pendingHandling:
			t.Fatalf("unexpected handle of events")
		default:
		}
		clock.Advance(a.conf.MaxDuration).MustWait(ctx)
		<-actionComplete

		assert.Equal(t, 2, total)
	})

	t.Run("reaching the max buffer size should invoke the on buffer full callback", func(t *testing.T) {
		called := false
		a := startNew[int](ctx, Config[int]{
			MaxDuration: DisableTimeLimit,
			MaxCount:    ImmediateDelivery,
			Handler: func(events []int) {
				total += len(events)
				actionComplete <- struct{}{}
			},
			QueueSize: 2,
			OnQueueFull: func([]int) {
				called = true
			},
		}, clock)

		a.OnEvent(1)
		assert.False(t, called)
		a.OnEvent(2)
		assert.False(t, called)

		a.OnEvent(3)
		assert.True(t, called)
	})
}

func TestAggregator_OnEventEdgeCases(t *testing.T) {
	t.Run("if configured with max count 1 event then the event should be sent immediately", func(t *testing.T) {
		total := 0
		actionComplete := make(chan struct{})
		a := startNew[int](context.Background(), Config[int]{
			MaxDuration: DisableTimeLimit,
			MaxCount:    ImmediateDelivery,
			Handler: func(events []int) {
				total += len(events)
				actionComplete <- struct{}{}
			},
			QueueSize: 1,
			OnQueueFull: func([]int) {
				t.Fatalf("unexpected buffer full")
			},
		}, quartz.NewMock(t))
		a.OnEvent(1)
		<-actionComplete
	})

	t.Run("if configured with max count 0 events then the events should be buffered indefinitely", func(t *testing.T) {
		a := startNew[int](context.Background(), Config[int]{
			MaxDuration: time.Nanosecond,
			MaxCount:    DisableCountLimit,
			Handler: func(events []int) {
				t.Fatalf("unexpected execution of action")
			},
			QueueSize: 1,
			OnQueueFull: func([]int) {
				t.Fatalf("unexpected buffer full")
			},
		}, quartz.NewMock(t))

		for i := 0; i < 1_000; i++ {
			a.OnEvent(1)
		}
		assert.Equal(t, 1_000, len(a.bufferedEvents))
	})

	t.Run("waiting a portion of max duration between two withstanding events should result in a flush only after the remaining portion of the max duration", func(t *testing.T) {
		ctx := context.Background()
		total := 0
		actionComplete := make(chan struct{})
		clock := quartz.NewMock(t)
		a := startNew[int](ctx, Config[int]{
			MaxDuration: time.Second,
			MaxCount:    10,
			Handler: func(events []int) {
				total += len(events)
				actionComplete <- struct{}{}
			},
			QueueSize: 1,
			OnQueueFull: func([]int) {
				t.Fatalf("unexpected buffer full")
			},
		}, clock)

		// first event is buffered
		a.OnEvent(1)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 0, total)

		// wait for the first portion, no flush expected
		clock.Advance(a.conf.MaxDuration / 2).MustWait(ctx)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 0, total)

		// second event is buffered
		a.OnEvent(2)
		assert.Equal(t, 2, len(a.bufferedEvents))
		assert.Equal(t, 0, total)

		// wait for the second portion, flush expected
		clock.Advance(a.conf.MaxDuration / 2).MustWait(ctx)
		<-actionComplete

		assert.Equal(t, 0, len(a.bufferedEvents))
		assert.Equal(t, 2, total)
	})

	t.Run("mixing time limit and count limit respects the time limit properly for deliver and non delivery", func(t *testing.T) {
		ctx := context.Background()
		total := 0
		actionComplete := make(chan struct{})
		clock := quartz.NewMock(t)
		a := startNew[int](ctx, Config[int]{
			MaxDuration: time.Second,
			MaxCount:    2,
			Handler: func(events []int) {
				total += len(events)
				actionComplete <- struct{}{}
			},
			QueueSize: 1,
			OnQueueFull: func([]int) {
				t.Fatalf("unexpected buffer full")
			},
		}, clock)

		// first event is buffered
		a.OnEvent(1)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 0, total)

		// wait portion: 25%
		clock.Advance(a.conf.MaxDuration / 4).MustWait(ctx)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 0, total)

		// events should be delivered due to count limit
		a.OnEvent(2)
		<-actionComplete
		assert.Equal(t, 0, len(a.bufferedEvents))
		assert.Equal(t, 2, total)

		// wait portion: 25%
		clock.Advance(a.conf.MaxDuration / 4).MustWait(ctx)

		// add 3 as our first event
		a.OnEvent(3)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 2, total)

		// wait remaining portion
		// event should not be delivered
		clock.Advance(a.conf.MaxDuration / 2).MustWait(ctx)
		assert.Equal(t, 1, len(a.bufferedEvents))
		assert.Equal(t, 2, total)

		// wait what for the remaining portion since event 3
		// now should be delivered
		clock.Advance(a.conf.MaxDuration / 2).MustWait(ctx)
		<-actionComplete
		assert.Equal(t, 0, len(a.bufferedEvents))
		assert.Equal(t, 3, total)
	})
}

func TestAggregator_consumeUnsafe(t *testing.T) {
	a := startNew[int](context.Background(), Config[int]{
		MaxDuration: time.Second,
		MaxCount:    5,
	}, quartz.NewMock(t))

	expectedCap := cap(a.bufferedEvents)
	a.bufferedEvents = append(a.bufferedEvents, 1, 2)
	cpy := a.consumeUnsafe()
	assert.Equal(t, cpy, []int{1, 2})
	a.bufferedEvents = append(a.bufferedEvents, 3)
	assert.Equal(t, a.bufferedEvents, []int{3})
	assert.Equal(t, expectedCap, cap(a.bufferedEvents))
}

func TestAggregator_stress(t *testing.T) {
	t.Skip()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := 0
	a := StartNew[int](ctx, Config[int]{
		MaxDuration: 1 * time.Second,
		MaxCount:    11,
		Handler: func(events []int) {
			received += len(events)
		},
		QueueSize: 10_000,
		OnQueueFull: func([]int) {
			t.Fatalf("buffer full")
		},
	})

	expected := a.conf.QueueSize
	wg := sync.WaitGroup{}
	wg.Add(expected)
	for i := 0; i < expected; i++ {
		go func() {
			if rand.Intn(500) == 0 {
				time.Sleep(time.Millisecond * 1000)
			}
			a.OnEvent(1)
			wg.Done()
		}()
	}

	wg.Wait()
	time.Sleep(time.Second * 3)
	assert.Equal(t, expected, received)
}
