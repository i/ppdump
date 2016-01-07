ppdump
========
set triggers for pprof dumps

how to get
------------

    go get github.com/i/ppdump

how to do
---------

```go
package main

import (
  "fmt"
  "time"

  "github.com/i/ppdump"
)

func main() {
  // use the default ppdump client
  ppdump.Start(ppdump.Config{
		Interval:  time.Second / 4,
		HardLimit: 500,   // trigger a dump when there are more than 500 goroutines
		Path:      "./",
		Profiles: map[string]int{
			"goroutine": 2, // dump the goroutine profile with debug level 2
		},
  })
  defer ppdump.Stop()

  // trigger a dump
  for i := 0; i < 501; i++ {
    go time.Sleep(time.Second * 5)
  }

  time.Sleep(time.Second) // give ppdump some time to check goroutines
}

```
