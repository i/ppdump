package main

import (
	"time"

	"github.com/i/ppdump"
)

func main() {
	// use the default ppdump client
	ppdump.Start(ppdump.Config{
		Interval:  time.Second / 4,
		HardLimit: 500, // trigger a dump when there are more than 500 goroutines
		Path:      "./pprof",
		Profiles: map[string]int{
			"goroutine": 1, // dump the goroutine profile with debug level 1
		},
	})
	defer ppdump.Stop()

	// trigger a dump
	for i := 0; i < 501; i++ {
		go time.Sleep(time.Second * 5)
	}

	time.Sleep(time.Second) // give ppdump some time to check goroutines
}
