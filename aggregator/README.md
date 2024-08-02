# Aggregator

An aggregator is used to aggregate events of any type until either the _time limit_ or the _count limit_ is reached.
It supports configuring a non-blocking behavior and may be combined with a static pool for increased concurrency.

## Usage

### Configuration
- `MaxDuration`: Maximum duration for an event to wait for delivery.
- `MaxCount`: Maximum number of events to be buffered.
- `Handler`: Callback function for handling aggregated events.
- `QueueSize`: Maximum number of aggregations that can be queued for handling.
- `OnQueueFull`: Optional callback function to be called when the queue is full.

### Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/gavraz/aggregator"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := aggregator.Config[int]{
		MaxDuration: 5 * time.Second,
		MaxCount:    10,
		Handler: func(events []int) {
			fmt.Println("Handling events:", events)
		},
		QueueSize: 5,
		OnQueueFull: func(events []int) {
			fmt.Println("Queue full, dropping events:", events)
		},
	}

	agg := aggregator.StartNew(ctx, conf)

	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(event int) {
			agg.OnEvent(event)
			wg.Done()
		}(i)
	}

	wg.Wait()
}

```