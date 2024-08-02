# Future

A Future is a basic primitive for managing future results of any type. The result can be set once but retrieved multiple times.

## Example 
```go
package main

import (
	"fmt"
	"github.com/gavraz/future"
	"time"
)

func main() {
	f := future.New[int]()

	go func() {
		time.Sleep(2 * time.Second)
		f.SetResult(42)
	}()

	go func() {
		select {
		// ...
		case <-f.C():
		}
		fmt.Println("Done waiting for result")
	}()

	fmt.Println("Result:", f.WaitResult())
}

```