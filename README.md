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
	ppdump.Start(ppdump.Config{
		PollInterval: time.Second,
		Throttle:     time.Minute,
		Profiles: map[string]ppdump.ProfileOpts{
			"goroutine": {
				Threshold: 10000,
				Action:    dumpToDisk,
			},
		},
	})

  // trigger a dump
  for i := 0; i < 501; i++ {
    go time.Sleep(time.Second * 5)
  }

  time.Sleep(time.Second) // give ppdump some time to check goroutines
}

func dumpToDisk(p *pprof.Profile) {
    f, err := os.Create(fmt.Sprintf("%s-%d.dump", p.Name(), time.Unix()))
    if err != nil {
        // handle err
    }
    defer f.Close()
    if _, err := p.WriteTo(f, 0); err != nil {
        // handle err
    }
}

```
