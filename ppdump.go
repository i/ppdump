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

type Config struct {
	Interval  time.Duration  // how often to poll for goroutines
	Throttle  time.Duration  // time to wait before writing another dump
	HardLimit int            // number of goroutines to trigger a dump
	Path      string         // path to write profiles to
	Profiles  map[string]int // map of profile names to debug level
}

// Start begins the
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
	sync.Mutex
	interval  time.Duration
	throttle  time.Duration
	hardLimit int
	path      string
	profiles  map[string]int
	stop      chan struct{}
	lastDump  time.Time
}

func NewDumper(c Config) *Dumper {
	return &Dumper{
		interval:  c.Interval,
		throttle:  c.Throttle,
		hardLimit: c.HardLimit,
		path:      c.Path,
		profiles:  c.Profiles,
	}
}

func (d *Dumper) Start() {
	d.Lock()
	defer d.Unlock()

	d.stop = make(chan struct{})

	if d.interval == 0 {
		d.interval = DefaultInterval
	}
	go d.runLoop()
}

func (d *Dumper) runLoop() {
	t := time.NewTicker(d.interval)
	for {
		select {
		case <-t.C:
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

	now := time.Now()
	if now.Before(d.lastDump.Add(d.throttle)) {
		return
	}
	d.lastDump = now

	for profile, debug := range d.profiles {
		p := pprof.Lookup(profile)
		if p == nil {
			continue
		}

		fname := fmt.Sprintf("%d.%s", now.Unix(), profile)
		f, err := os.Create(path.Join(d.path, fname))
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
	return runtime.NumGoroutine() >= d.hardLimit
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
