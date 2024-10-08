package aggregator

import (
	"context"
	"github.com/coder/quartz"
	"sync"
	"time"
)

const (
	// DisableTimeLimit can be passed as Config.MaxDuration to disable the time limit.
	DisableTimeLimit = maxDuration
	// DisableCountLimit can be passed as Config.MaxCount to disable the count limit.
	DisableCountLimit = 0
	// ImmediateDelivery can be passed as Config.MaxCount along with DisableTimeLimit to enable immediate delivery.
	ImmediateDelivery = 1

	// Why do we set the ticker to the maximum duration?
	// When there are no buffered events, there's no need for the serving routine to handle ticks.
	// Therefore, we use a large value to ensure the ticker won't trigger.
	// We will reset the timer to Config.MaxDuration on the first event.
	maxDuration = time.Duration(1<<63 - 1)
)

type Config[T any] struct {
	// MaxDuration is the max duration for an event to wait for delivery.
	// This limit can be disabled using DisableTimeLimit.
	MaxDuration time.Duration
	// MaxCount is the max number of events to be buffered.
	// This limit can be disabled using DisableCountLimit.
	// A value of indicates no limit on buffering while a limit of 1 indicates immediate event delivery.
	MaxCount int
	// Handler is the callback for handling aggregated events.
	Handler func(events []T)
	// QueueSize represents max number of aggregations that can be queued for handling.
	QueueSize int
	// OnQueueFull is an optional callback to be called when the queue is full.
	OnQueueFull func(events []T)
}

// Aggregator is used to aggregate events of any type.
// Events are accumulated until either the time limit specified by Config.MaxDuration
// or the count limit specified by Config.MaxCount is reached.
// When a limit is reached, the events are processed using the handler defined in Config.Handler.
// To ensure non-blocking handling of events, Config.QueueSize can be set, which is useful if
// event handling can be slower than event generation. Optionally, a Config.OnQueueFull callback
// can be specified to handle situations when the buffer is full.
type Aggregator[T any] struct {
	conf Config[T]

	clock           quartz.Clock
	pendingHandling chan []T
	cancelled       <-chan struct{}

	mu             sync.Mutex
	bufferedEvents []T
	ticker         *quartz.Ticker
}

// StartNew initiates and starts a new aggregator for the provided type and configurations.
func StartNew[T any](ctx context.Context, conf Config[T]) *Aggregator[T] {
	clock := quartz.NewReal()
	return startNew(ctx, conf, clock)
}

func startNew[T any](ctx context.Context, conf Config[T], clock quartz.Clock) *Aggregator[T] {
	start := NewStarter[T](conf, clock)
	return start(ctx)
}

// NewStarter initiates an aggregator and returns a starter function to start and get the aggregator.
func NewStarter[T any](conf Config[T], clock quartz.Clock) (start func(ctx context.Context) *Aggregator[T]) {
	if conf.MaxDuration == 0 {
		panic("bad config: max duration cannot be zero")
	}

	a := &Aggregator[T]{
		conf:            conf,
		clock:           clock,
		pendingHandling: make(chan []T, conf.QueueSize),
		bufferedEvents:  make([]T, 0, conf.MaxCount),
		ticker:          clock.NewTicker(maxDuration),
	}

	return func(ctx context.Context) *Aggregator[T] {
		a.start(ctx)
		return a
	}
}

func (a *Aggregator[T]) start(ctx context.Context) {
	a.cancelled = ctx.Done()

	go func() {
		defer a.ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case events := <-a.pendingHandling:
				a.conf.Handler(events)
			case <-a.ticker.C:
				a.mu.Lock()
				pending := a.consumeUnsafe()
				a.mu.Unlock()
				if len(pending) > 0 {
					a.conf.Handler(pending)
				}
			}
		}
	}()
}

// OnEvent adds a new event to be processed by the aggregator.
func (a *Aggregator[T]) OnEvent(event T) {
	var consumedEvents []T

	a.mu.Lock()
	wasEmpty := len(a.bufferedEvents) == 0
	a.bufferedEvents = append(a.bufferedEvents, event)
	if a.conf.MaxCount > 0 && len(a.bufferedEvents) >= a.conf.MaxCount {
		consumedEvents = a.consumeUnsafe()
	}
	notEmpty := len(a.bufferedEvents) > 0
	if notEmpty && wasEmpty {
		// this is the first time we see a withstanding event since our last flush
		// we reset the ticker to start the countdown for the next delivery
		a.ticker.Reset(a.conf.MaxDuration)
	}
	a.mu.Unlock()

	if len(consumedEvents) > 0 {
		a.handle(consumedEvents)
	}
}

// QueueLen returns the number of aggregations that are queued for handling.
func (a *Aggregator[T]) QueueLen() int {
	return len(a.pendingHandling)
}

func (a *Aggregator[T]) consumeUnsafe() []T {
	consumed := a.bufferedEvents
	a.bufferedEvents = make([]T, 0, a.conf.MaxCount)
	return consumed
}

func (a *Aggregator[T]) handle(events []T) {
	select {
	case a.pendingHandling <- events:
	case <-a.cancelled:
	default:
		if onFull := a.conf.OnQueueFull; onFull != nil {
			onFull(events)
		}
	}
}
