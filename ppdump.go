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

var std *Dumper

func Start(c Config) {
	std = NewDumper(c)
	std.Start()
}

func Stop() {
	if std != nil {
		std.Stop()
	}
}

type Dumper struct {
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

type Config struct {
	// how often to poll for goroutines
	Interval   time.Duration
	BufferSize int //
	HardLimit  int
	Path       string
	Profiles   map[string]int
}

func NewDumper(c Config) *Dumper {
	return &Dumper{
		Interval:   c.Interval,
		BufferSize: c.BufferSize,
		HardLimit:  c.HardLimit,
		Path:       c.Path,
		Profiles:   c.Profiles,
	}
}

func (d *Dumper) Start() {
	d.Lock()
	defer d.Unlock()

	d.stop = make(chan struct{})

	if d.Interval == 0 {
		d.Interval = DefaultInterval
	}
	go d.runLoop()
}

func (d *Dumper) runLoop() {
	for {
		select {
		case <-time.Tick(d.Interval):
			if d.thresholdExceeded() {
				d.dump()
			}
		case <-d.stop:
			return
		}
	}
}

func (d *Dumper) dump() {
	d.Lock()
	defer d.Unlock()

	now := time.Now().Unix()
	for profile, debug := range d.Profiles {
		p := pprof.Lookup(profile)
		if p == nil {
			continue
		}

		fname := fmt.Sprintf("%d-%s", now, profile)
		f, err := os.Create(path.Join(d.Path, fname))
		if err != nil {
			continue
		}
		defer f.Close()
		p.WriteTo(f, debug)
	}
}

// Stops the runloop. No-op if called more than once.
func (d *Dumper) Stop() {
	d.Lock()
	defer d.Unlock()
	defer func() { recover() }() // doesn't matter if the channel is closed twice
	close(d.stop)
}

func (d *Dumper) thresholdExceeded() bool {
	return runtime.NumGoroutine() >= d.HardLimit
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
