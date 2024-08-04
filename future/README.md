# Future

A Future is a basic primitive for managing future results of any type. The result can be set once but retrieved multiple times.

## Example 
```go
package main

import (
	"fmt"
	"github.com/gavraz/async/future"
	"time"
)

func makePromise() *future.Future[int] {
	p := future.NewPromise[int]()

	go func() {
		time.Sleep(2 * time.Second)
		p.Set(42)
	}()

	return p.Future()
}

func main() {
	f := makePromise()

	go func() {
		select {
		// ...
		case <-f.Done():
		}
		fmt.Println("Done waiting for result")
	}()

	fmt.Println("Result:", f.Value())
}


```