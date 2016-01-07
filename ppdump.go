package ppdump

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

const (
	DefaultBufferSize = 1000
	DefaultInterval   = time.Second
)

var std *A

func Start(opts NewAOpts) {
	std = NewA(opts)
	std.Start()
}

func Stop() {
	if std != nil {
		std.Stop()
	}
}

type A struct {
	Interval   time.Duration
	BufferSize int
	HardLimit  int
	Path       string
	Debug      int
	Profiles   map[string]int

	buf buffer
	sync.Mutex
	stop chan struct{}
}

type NewAOpts struct {
	// how often to poll for goroutines
	Interval   time.Duration
	BufferSize int //
	HardLimit  int
	Path       string
	Profiles   map[string]int
}

func NewA(opts NewAOpts) *A {
	return &A{
		Interval:   opts.Interval,
		BufferSize: opts.BufferSize,
		HardLimit:  opts.HardLimit,
		Path:       opts.Path,
		Profiles:   opts.Profiles,
	}
}

func (a *A) Start() {
	a.Lock()
	defer a.Unlock()

	a.stop = make(chan struct{})

	if a.Interval == 0 {
		a.Interval = DefaultInterval
	}
	go a.runLoop()
}

func (a *A) runLoop() {
	for {
		select {
		case <-time.Tick(a.Interval):
			if a.thresholdExceeded() {
				a.dump()
			}
		case <-a.stop:
			return
		}
	}
}

func (a *A) dump() {
	a.Lock()
	defer a.Unlock()

	now := time.Now().Unix()
	for profile, debug := range a.Profiles {
		p := pprof.Lookup(profile)
		if p == nil {
			continue
		}

		fname := fmt.Sprintf("%d-%s", now, profile)
		f, err := os.Create(path.Join(a.Path, fname))
		if err != nil {
			continue
		}
		defer f.Close()
		p.WriteTo(f, debug)
	}
}

// Stops the runloop. No-op if called more than once.
func (a *A) Stop() {
	a.Lock()
	defer a.Unlock()
	defer func() { recover() }() // doesn't matter if the channel is closed twice
	close(a.stop)
}

func (a *A) thresholdExceeded() bool {
	return runtime.NumGoroutine() >= a.HardLimit
}

// type to buffer previous results
type buffer []int

func (b *buffer) push(n int) {
	*b = append(*b, n)
}

func (b *buffer) shift(n int) {
	*b = (*b)[:len(*b)-1]
}

func (b *buffer) insert(n int) {
	if len(*b) < cap(*b) {
		b.push(n)
	} else {
		b.shift(n)
	}
}
