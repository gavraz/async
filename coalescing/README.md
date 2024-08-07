# Coalescing Queue
A coalescing queue allows multiple operations to be combined into a single operation. This is useful when a heavy
operation might be executed multiple times, but only needs to be performed once if it hasn't started yet.
For instance, if an operation (OP) is currently being executed and a series of additional requests to execute OP are received
(e.g., {OP, OP, OP}), these requests can be consolidated into a single operation ({op}).
## Usage
### Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/gavraz/async/coalescing"
	"time"
)

func printHello(_ context.Context) {
	fmt.Println("Hello")
	time.Sleep(2 * time.Second)
}

func main() {
	ctx := context.Background()
	execute := coalescing.Queue(ctx, printHello)

	execute(ctx)         // first is executed 
	execute(ctx)         // second will wait for first
	done := execute(ctx) // third will be dropped
	<-done               // will return after the second execution completes

	// output:
	// Hello
	// Hello
}

```