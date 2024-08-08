package aggregator

import (
	"fmt"
	"time"
)

type Config[T any] struct {
	// MaxDuration is the max duration for an event to wait for delivery.
	// This limit can be disabled using DisableTimeLimit.
	MaxDuration time.Duration
	// MaxBufferedEvents is the max number of events to be buffered.
	// This limit can be disabled using DisableCountLimit.
	// A value of 0 indicates no limit on buffering while a limit of 1 indicates immediate event delivery.
	MaxBufferedEvents int
	// Handler is the callback for handling aggregated events.
	Handler func(events []T)
	// QueueSize represents max number of aggregations that can be queued for handling.
	QueueSize int
	// OnQueueFull is an optional callback to be called when the queue is full.
	OnQueueFull func(events []T)
}

func (c *Config[T]) Validate() error {
	if c.MaxDuration <= 0 {
		return fmt.Errorf("MaxDuration must be greater than zero")
	}
	if c.MaxBufferedEvents < 0 {
		return fmt.Errorf("MaxBufferedEvents must be greater than or equal to zero")
	}
	if c.QueueSize <= 0 || c.QueueSize&(c.QueueSize-1) != 0 {
		return fmt.Errorf("QueueSize must be a power of 2")
	}
	if c.Handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	return nil
}
